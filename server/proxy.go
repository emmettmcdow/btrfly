package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/emmettmcdow/kache/server/kache"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

/*
States:
  - Recording / Playback / Standby   - R/P/S
  - Kache Present / Not Present      - KP/KN
  - Upstream Exists / Does Not Exist - UE/UD
  - Upstream New / Old               - UN/UO

Expected Behavior For Each State:
|
|
|---- R
|     |---- KP
|     |     |---- UE
|     |     |     |---- UN - Use Upstream and update Kache artifact
|     |     |     |
|     |     |     |---- UO - Use kached
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
type proxy_mode uint8

const MODE_R proxy_mode = 0
const MODE_P proxy_mode = 1
const MODE_S proxy_mode = 2

var ProxyMode = MODE_S
var BuildTag = "shoop da woop"

// TODO: optimize the buffer, switch to byte slice
type tempResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func main() {
	var k kache.Handler

	// TODO: this is temporary for testing
	k = kache.CreateMemory()
	k.AddUser(kache.CreateUser())

	var httpClient *http.Client
	var currUser uint64 = 0

	log.Print("Starting Kache...")

	httpClient = init_custom_transport()

	m := http.NewServeMux()
	s := http.Server{Addr: ":80", Handler: m}
	m.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		full_url := r.Host + r.URL.String()
		log.Printf("Received a %s request to %s", r.Method, full_url)
		http.Error(w, "Shutting down", 400)
		s.Shutdown(context.Background())
	})
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		full_url := r.Host + r.URL.String()
		log.Printf("Received a %s request to %s", r.Method, full_url)
		// TODO: use the conditional get
		switch ProxyMode {
		case MODE_R:
			upstreamArtifact := &kache.Artifact{}

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
			cachedArtifact, err := k.GetArtifact(full_url, BuildTag, currUser)
			if err == nil && upstreamArtifact.Equal(cachedArtifact) { // If artifact already exists, just tag it
				// Tag the existing one
				// TODO: fix all the nonstandard names!
				k.TagArtifact(cachedArtifact, BuildTag, full_url, currUser)
			} else {
				err = k.AddArtifact(upstreamArtifact, full_url, BuildTag, currUser)
				if err != nil {
					log.Printf("Failed to add artifact to Kache: %s", err)
					http.Error(w,
						"Error creating proxy request",
						http.StatusInternalServerError)
				}
			}

		case MODE_P:
			cachedArtifact, err := k.GetArtifact(full_url, BuildTag, currUser)
			if err != nil {
				log.Printf("Failed to retrieve the requested artifact from Kache. "+
					"Something went seriously wrong.: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			} else {
				err = respondWithArtifact(w, r, cachedArtifact)
				if err != nil {
					log.Printf("Failed to send kached artifact: %s", err)
					http.Error(w,
						"Error creating proxy request",
						http.StatusInternalServerError)
				}
			}
			// err = formatUpstreamResponse(w, response)
			// if err != nil {
			// 	log.Printf("Failed to format the response from upstream: %s", err)
			// 	http.Error(response,
			// 		"Error creating proxy request",
			// 		http.StatusInternalServerError)
			// }
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
			log.Fatal("Kache mode is invalid!")
		}
	})
	log.Print(s.ListenAndServe())
	return
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

func respondWithArtifact(w http.ResponseWriter, r *http.Request, artifact *kache.Artifact) (err error) {
	// Copy the headers from the proxy response to the original response
	// for name, values := range src.Header {
	// 	for _, value := range values {
	// 		dest.Header().Add(name, value)
	// 	}
	// }

	w.Header().Add("Content-Length", fmt.Sprint(len(artifact.Data)))
	// w.Header().Add("Content

	// Set the status code of the original response to the status code of the proxy response
	w.WriteHeader(200)

	// Copy the body of the proxy response to the original response
	// TODO: Icky icky icky fixme pwease
	_, err = io.Copy(w, bytes.NewBuffer(artifact.Data))
	return err
}
