package main

import (
	"context"
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

func main() {
	var proxy_mode = MODE_S
	var build_id = ""
	var httpClient *http.Client
	var tempResponse http.ResponseWriter

	log.Print("Starting Kache...")

	httpClient = init_custom_transport()

	http.HandleFunc("/", func(response http.ResponseWriter, r *http.Request) {
		full_url := r.Host.String() + r.URL.String()
		log.Printf("Received a %s request to %s", r.Method, full_url)
		// TODO: use the conditional get
		switch proxy_mode {
		case MODE_R:
			artifact, err := kache.GetArtifact(full_url, build_id)
			if err != nil {
				var upstream_artifact []bytes
				err = relayRequest(tempResponse, r)
				if err == nil {
					// TODO: be more efficient?
					if upstream_artifact != artifact {
						kache.AddArtifact(artifact, full_url, build_id)
						if err != nil {
							log.Printf("Failed to add artifact to Kache: %s", err)
							http.Error(w,
								"Error creating proxy request",
								http.StatusInternalServerError)
						}
					}
				}
			} else {
				artifact, err = relayRequest(tempResponse, r)
				if err != nil {
					log.Printf("Failed to send request to upstream: %s", err)
					http.Error(w,
						"Error creating proxy request",
						http.StatusInternalServerError)
				}
				kache.AddArtifact(artifact, full_url, build_id)
				if err != nil {
					log.Printf("Failed to add artifact to Kache: %s", err)
					http.Error(w,
						"Error creating proxy request",
						http.StatusInternalServerError)
				}
			}
		case MODE_P:
			artifact, err := kache.GetArtifact(full_url, build_id)
			if !artifact || err {
				log.Printf("Failed to retrieve the requested artifact from Kache. "+
					"Something went seriously wrong.: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
			err = respondWithArtifact(tempResponse, r, artifact)
			if err != nil {
				log.Printf("Failed to send kached artifact: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
		case MODE_S:
			upstreamRequest, err := generateUpstreamRequest(r)
			if err != nil {
				log.Printf("Failed to generate an upstream request: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
			err := relayRequest(tempResponse, upstreamRequest, httpClient)
			if err != nil {
				log.Printf("Failed to send request to upstream: %s", err)
				http.Error(w,
					"Error creating proxy request",
					http.StatusInternalServerError)
			}
			err := formatUpstreamResponse(w, tempResponse)
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
	log.Fatal(http.ListenAndServe(":80", nil))
}

func init_custom_transport() {
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
}

func generateUpstreamRequest(r *http.Request) (proxyReq *http.Request) {
	// Create a new HTTP request with the same method, URL, and body as the original request
	targetURL := "http://" + r.Host + r.URL.String()
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		return err
	}

	// Copy the headers from the original request to the proxy request
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	return proxyReq
}

func formatUpstreamResponse(dest http.ResponseWriter, src http.Response) {
	// Copy the headers from the proxy response to the original response
	for name, values := range src.Header {
		for _, value := range values {
			dest.Header().Add(name, value)
		}
	}

	// Set the status code of the original response to the status code of the proxy response
	dest.WriteHeader(src.StatusCode)

	// Copy the body of the proxy response to the original response
	_, err = io.Copy(dest, src.Body)
	return err
}

func relayRequest(w http.ResponseWriter, r *http.Request, httpClient *httpClient) (err error) {
	// Send the proxy request using the custom transport
	var resp http.Response
	resp, err = httpClient.Do(proxyReq)
	if err != nil {
		return err
	}
	defer resp.Body.close()
	// Copy the headers from the proxy response to the original response
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set the status code of the original response to the status code of the proxy response
	w.WriteHeader(resp.StatusCode)

	// Copy the body of the proxy response to the original response
	_, err = io.Copy(w, resp.Body)
	return err
}

func respondWithArtifact(w http.ResponseWriter, r *http.Request, httpClient *httpClient) {
	return
}
