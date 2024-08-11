package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/emmettmcdow/btrfly/server/cache"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

/*
States:
  - Recording / Playback / Standby   - R/P/S
  - btrfly Present / Not Present      - KP/KN
  - Upstream Exists / Does Not Exist - UE/UD
  - Upstream New / Old               - UN/UO

Expected Behavior For Each State:
|
|
|---- R
|     |---- KP
|     |     |---- UE
|     |     |     |---- UN - Use Upstream and update cached artifact
|     |     |     |
|     |     |     |---- UO - Use cached
|     |     |
|     |     |---- UD - Pass along the upstream 400
|     |
|     |---- KN
|           |---- UE - Use upstream and create entry
|           |
|           |---- UD - Pass along the upstream 400
|
|---- P
|     |---- KP - Give user what they want
|     |
|     |---- KN - Something has gone critically wrong
|
|---- S - Behave as if there is no proxy or kaching. Just passthrough
*/
type ProxyMode uint8

const (
	MODE_R ProxyMode = iota
	MODE_P
	MODE_S
)

func (m ProxyMode) String() string {
	switch m {
	case MODE_R:
		return "Record"
	case MODE_P:
		return "Playback"
	case MODE_S:
		return "Standby"
	default:
		return ""
	}
}

var proxyMode = MODE_S
var buildTag = "shoop da woop"
var currUser uint64 = 0

type tempResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func proxy(wg *sync.WaitGroup, port uint, tlsEnabled bool) (s *http.Server) {
	var k cache.Handler
	var config *tls.Config

	// TODO: this is temporary for testing
	k = cache.CreateMemory()
	k.AddUser(cache.CreateUser())

	var httpClient *http.Client

	log.Print("Starting btrfly...")

	httpClient = init_custom_transport()

	m := http.NewServeMux()
	if tlsEnabled {
		cert, err := tls.LoadX509KeyPair("server.pem", "server.key")
		if err != nil {
			fmt.Printf("Failed to load certificate keypair: %s\n", err)
		}
		config = &tls.Config{Certificates: []tls.Certificate{cert}}
	}
	s = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: m, TLSConfig: config}
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		full_url := r.Host + r.URL.String()
		log.Printf("Received a %s request to %s", r.Method, full_url)
		// TODO: use the conditional get
		switch proxyMode {
		case MODE_R:
			upstreamArtifact := &btrfly.Artifact{}

			upstreamRequest, err := generateUpstreamRequest(r)
			if err != nil {
				log.Printf("Failed to generate an upstream request: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
			response, err := relayRequest(upstreamRequest, httpClient)
			if err != nil {
				log.Printf("Failed to relay request to upstream: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
			n, err := upstreamArtifact.Write(response.Body)
			if n != len(response.Body) {
				log.Printf("Failed to copy over http response body to artifact: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}

			err = formatUpstreamResponse(w, response)
			if err != nil {
				log.Printf("Failed to format the response from upstream: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
			cachedArtifact, err := k.GetArtifact(full_url, buildTag, currUser)
			if err == nil && upstreamArtifact.Equal(cachedArtifact) { // If artifact already exists, just tag it
				// Tag the existing one
				// TODO: fix all the nonstandard names!
				k.TagArtifact(cachedArtifact, buildTag, full_url, currUser)
			} else {
				err = k.AddArtifact(upstreamArtifact, full_url, buildTag, currUser)
				if err != nil {
					log.Printf("Failed to add artifact to btrfly: %s", err)
					http.Error(w,
						"Error creating proxy request",
						http.StatusInternalServerError)
				}
			}

		case MODE_P:
			cachedArtifact, err := k.GetArtifact(full_url, buildTag, currUser)
			if err != nil {
				log.Printf("Failed to retrieve the requested artifact from btrfly. "+
					"Something went seriously wrong.: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			} else {
				err = respondWithArtifact(w, r, cachedArtifact)
				if err != nil {
					log.Printf("Failed to send cached artifact: %s", err)
					http.Error(w,
						"Error creating proxy request",
						http.StatusInternalServerError)
				}
			}
		case MODE_S:
			upstreamRequest, err := generateUpstreamRequest(r)
			if err != nil {
				log.Printf("Failed to generate an upstream request: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
			response, err := relayRequest(upstreamRequest, httpClient)
			if err != nil {
				log.Printf("Failed to send request to upstream: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
			err = formatUpstreamResponse(w, response)
			if err != nil {
				log.Printf("Failed to format the response from upstream: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}

		default:
			log.Fatal("btrfly mode is invalid!")
		}
	})
	go func() {
		defer wg.Done()
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe failed: %s\n", err)
		}
	}()

	return s
}

func Login(ID string) (err error) {
	m, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		return fmt.Errorf("Failed to convert ID %s to integer: %s", ID, err)
	}
	currUser = m
	return nil
}

func Tag(tag string) (err error) {
	buildTag = tag
	return nil
}

func Mode(mode string) (err error) {
	m, err := strconv.ParseUint(mode, 10, 8)
	if err != nil {
		return fmt.Errorf("Failed to convert mode %s to integer: %s", mode, err)
	}
	if m > 2 {
		return fmt.Errorf("Invalid error mode %d", m)
	}
	proxyMode = ProxyMode(m)
	return nil
}

func init_custom_transport() (httpClient *http.Client) {
	var (
		dnsResolverIP        = "8.8.8.8:53" // Google DNS resolver.
		dnsResolverProto     = "udp"        // Protocol to use for the DNS resolver
		dnsResolverTimeoutMs = 5000         // Timeout (ms) for the DNS resolver (optional)
	)

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
				}
				return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	http.DefaultTransport.(*http.Transport).DialContext = dialContext
	httpClient = &http.Client{}

	return httpClient
}

func generateUpstreamRequest(r *http.Request) (proxyReq *http.Request, err error) {
	// Create a new HTTP request with the same method, URL, and body as the original request
	targetURL := "http://" + r.Host + r.URL.String()
	proxyReq, err = http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		return nil, err
	}

	// Copy the headers from the original request to the proxy request
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	return proxyReq, err
}

func formatUpstreamResponse(dest http.ResponseWriter, src tempResponse) (err error) {
	// Copy the headers from the proxy response to the original response
	for name, values := range src.Header {
		for _, value := range values {
			dest.Header().Add(name, value)
		}
	}

	// Set the status code of the original response to the status code of the proxy response
	dest.WriteHeader(src.StatusCode)

	// Copy the body of the proxy response to the original response
	// TODO: Icky icky icky fixme pwease
	_, err = io.Copy(dest, bytes.NewBuffer(src.Body))
	return err
}

type clientSender interface {
	Do(r *http.Request) (*http.Response, error)
}

func relayRequest(proxyReq *http.Request, httpClient clientSender) (response tempResponse, err error) {
	// Send the proxy request using the custom transport
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()
	response.Header = http.Header{}
	// Copy the headers from the proxy response to the original response
	for name, values := range resp.Header {
		for _, value := range values {
			response.Header.Add(name, value)
		}
	}

	// Set the status code of the original response to the status code of the proxy response
	response.StatusCode = resp.StatusCode

	// Copy the body of the proxy response to the original response
	body, err := io.ReadAll(resp.Body)
	response.Body = body
	return response, err
}

func respondWithArtifact(w http.ResponseWriter, r *http.Request, artifact *btrfly.Artifact) (err error) {
	w.Header().Add("Content-Length", fmt.Sprint(len(artifact.Data)))

	// Set the status code of the original response to the status code of the proxy response
	w.WriteHeader(200)

	// Copy the body of the proxy response to the original response
	// TODO: Icky icky icky fixme pwease
	_, err = io.Copy(w, bytes.NewBuffer(artifact.Data))
	return err
}
