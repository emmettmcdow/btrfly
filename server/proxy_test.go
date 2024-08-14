package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

const port = 8081

// ************************************************************** Integration Tests

func doBtrflyRequest(method string, URL string, client *http.Client) (_ string, _ int, err error) {

	req, err := http.NewRequest(method, URL, http.NoBody)
	if err != nil {
		return "", 0, fmt.Errorf("failed to make new req: %s", err)
	}
	// Doing the part of the btrfly client here - Redirecting to proxy
	newURL, err := url.Parse(strings.ReplaceAll(URL, "1234", fmt.Sprint(port)))
	if err != nil {
		return "", 0, fmt.Errorf("failed to redirect req: %s", err)
	}
	req.URL = newURL
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to \"Do\": %s", err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to copy Body: %s", err)
	}
	return string(body), resp.StatusCode, err
}

func TestProxyRecordAndPlayback(t *testing.T) {

	httpClient := http.DefaultClient
	serverReady := make(chan func() (err error))

	aOriginal := "this is the /root/a file"
	bOriginal := "this is the /root/b file"
	cOriginal := "this is the /root/c file"

	aUpdated := "Updated /root/a file"
	bUpdated := "Updated /root/b file"
	// cUpdated := "Updated /root/c file"

	memoryFS := fstest.MapFS{
		"root/a": {Data: []byte(aOriginal)},
		"root/b": {Data: []byte(bOriginal)},
		"root/c": {Data: []byte(cOriginal)},
	}

	// btrfly
	wg := &sync.WaitGroup{}
	wg.Add(1)

	s := proxy(wg, port, false)
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer func() {
		err := s.Shutdown(timeout)
		if err != nil {
			fmt.Printf("failed to shutdown s: %s", err)
		}
	}()

	// Upstream Server
	go func() {
		l, err := net.Listen("tcp", ":1234")
		if err != nil {
			fmt.Printf("Listener failed: %s\n", err)
		}
		server := &http.Server{}
		server.Handler = http.FileServerFS(memoryFS)
		// Signal that server is open for business.
		serverReady <- server.Close

		err = server.Serve(l)
		if err != nil {
			fmt.Printf("Failed to Serve  with error: %s\n", err)
		}
		fmt.Println("Shutting down fileserver")
	}()
	shutdown := <-serverReady
	defer func() {
		if err := shutdown(); err != nil {
			fmt.Printf("Failed to shutdown: %s\n", err)
		}
	}()

	// Set to record
	proxyMode = MODE_R

	t.Run("RECORD GET http://127.0.0.1:1234/root/a ORIGINAL", func(t *testing.T) {
		body, statusCode, err := doBtrflyRequest("GET", "http://127.0.0.1:1234/root/a", httpClient)
		if err != nil {
			t.Errorf("Failed to do http request: %s\n", err)
		}
		if statusCode != 200 {
			t.Errorf("statusCode: got %d, want: 200", statusCode)
		}
		if body != aOriginal {
			t.Errorf("Response:\n    got: %s\n    want: %s\n", body, aOriginal)
		}
	})

	t.Run("RECORD GET http://127.0.0.1:1234/root/b ORIGINAL", func(t *testing.T) {
		// TODO: what should be done if a user downloads the same resource twice in one build?
		body, statusCode, err := doBtrflyRequest("GET", "http://127.0.0.1:1234/root/b", httpClient)
		if err != nil {
			t.Errorf("Failed to do http request: %s\n", err)
		}
		if statusCode != 200 {
			t.Errorf("statusCode: got %d, want: 200", statusCode)
		}
		if body != bOriginal {
			t.Errorf("Response:\n    got: %s\n    want: %s\n", body, bOriginal)
		}
	})

	// Set to playback
	proxyMode = MODE_P

	// Update the filesystem
	memoryFS["root/a"] = &fstest.MapFile{Data: []byte(aUpdated)}
	memoryFS["root/b"] = &fstest.MapFile{Data: []byte(bUpdated)}

	t.Run("PLAYBACK GET http://127.0.0.1:1234/root/a UPDATED", func(t *testing.T) {
		body, statusCode, err := doBtrflyRequest("GET", "http://127.0.0.1:1234/root/a", httpClient)
		if err != nil {
			t.Errorf("Failed to do http request: %s\n", err)
		}
		if statusCode != 200 {
			t.Errorf("statusCode: got %d, want: 200", statusCode)
		}
		if body != aOriginal {
			t.Errorf("Response:\n    got: %s\n    want: %s\n", body, aOriginal)
		}
	})

	t.Run("PLAYBACK GET http://127.0.0.1:1234/root/b UPDATED", func(t *testing.T) {
		// TODO: what should be done if a user downloads the same resource twice in one build?
		body, statusCode, err := doBtrflyRequest("GET", "http://127.0.0.1:1234/root/b", httpClient)
		if err != nil {
			t.Errorf("Failed to do http request: %s\n", err)
		}
		if statusCode != 200 {
			t.Errorf("statusCode: got %d, want: 200", statusCode)
		}
		if body != bOriginal {
			t.Errorf("Response:\n    got: %s\n    want: %s\n", body, bOriginal)
		}
	})

	t.Run("PLAYBACK GET http://127.0.0.1:1234/root/c DNE", func(t *testing.T) {
		_, statusCode, err := doBtrflyRequest("GET", "http://127.0.0.1:1234/root/c", httpClient)
		if err != nil {
			t.Errorf("Failed to do http request: %s\n", err)
		}
		if statusCode >= 200 && statusCode < 300 {
			t.Errorf("Successfully got artifact that should not be possible. Got code %d", statusCode)
		}
	})

}

func TestPassthroughProxy(t *testing.T) {

	// Set to playback
	proxyMode = MODE_S

	httpClient := http.DefaultClient
	serverReady := make(chan func() (err error))

	memoryFS := fstest.MapFS{
		"root/a": {Data: []byte("this is the /root/a file")},
		"root/b": {Data: []byte("this is the /root/b file")},
		"root/c": {Data: []byte("this is the /root/c file")},
	}

	// btrfly
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s := proxy(wg, port, false)
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer func() {
		err := s.Shutdown(timeout)
		if err != nil {
			fmt.Printf("failed to shutdown s: %s", err)
		}
	}()

	// Upstream Server
	go func() {
		l, err := net.Listen("tcp", ":1234")
		if err != nil {
			fmt.Printf("Listener failed: %s\n", err)
		}
		server := &http.Server{}
		server.Handler = http.FileServerFS(memoryFS)
		// Signal that server is open for business.
		serverReady <- server.Close

		err = server.Serve(l)
		if err != nil {
			t.Errorf("Failed to serve with error: %s\n", err)
		}
		fmt.Println("Shutting down fileserver")
	}()
	shutdown := <-serverReady
	defer func() {
		if err := shutdown(); err != nil {
			fmt.Printf("Failed to shutdown: %s\n", err)
		}
	}()

	cases := []struct {
		Method   string
		Url      string
		Response struct {
			ResponseCode int
		}
	}{
		{"GET", "http://127.0.0.1:1234/root/a", struct{ ResponseCode int }{200}},
		{"GET", "http://127.0.0.1:1234/root/b", struct{ ResponseCode int }{200}},
		{"GET", "http://127.0.0.1:1234/root/c", struct{ ResponseCode int }{200}},
		{"GET", "http://127.0.0.1:1234/root/DNE", struct{ ResponseCode int }{404}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s %s - %d", tc.Method, tc.Url, tc.Response.ResponseCode), func(t *testing.T) {
			_, statusCode, err := doBtrflyRequest(tc.Method, tc.Url, httpClient)
			if err != nil {
				t.Errorf("Failed to do http request: %s\n", err)
			}
			if statusCode != tc.Response.ResponseCode {
				t.Errorf("Response Code - Got: %d, Want: %d\n", statusCode, tc.Response.ResponseCode)
			}
		})
	}
}

// ************************************************************** Unit Tests |
type DumbClient struct {
	Err error
}

func (d DumbClient) Do(r *http.Request) (response *http.Response, err error) {
	response = &http.Response{Header: http.Header{}}
	for name, values := range r.Header {
		for _, value := range values {
			response.Header.Add(name, value)
		}
	}

	response.StatusCode = 200

	tmpBod, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("SOmething went wrong")
	}
	response.Body = io.NopCloser(bytes.NewReader(tmpBod))

	return response, d.Err
}

// TODO: Anonymize the case structs - more idiomatic
type relayRequestCase struct {
	name        string
	req         *http.Request
	client      *clientSender
	want        tempResponse
	expectedErr error
}

func TestRelayRequest(t *testing.T) {
	var fakeClient clientSender

	stdHeader := http.Header{
		"Key1": {"1a", "1b", "1c"},
		"Key2": {"2a", "2b", "2c"},
		"Key3": {"3a", "3b", "3c"},
	}
	stdText := "Woohoo!\noh no\n WOOHOO! "
	stdBody := strings.NewReader(stdText)
	stdRequest, _ := http.NewRequest("GET", "http://google.com/a/b/c", stdBody)
	badRequest, _ := http.NewRequest("GET", "http://FAKE.com/not/real", stdBody)
	stdRequest.Header = stdHeader

	successResponse := tempResponse{StatusCode: 200, Header: stdHeader}
	successResponse.Body = []byte(stdText)
	stdBody.Reset(stdText)

	fakeClient = DumbClient{Err: nil}

	cases := []relayRequestCase{
		{"Fake client GET success", stdRequest, &fakeClient, successResponse, nil},
		{"Fake client GET failure", badRequest, &fakeClient, tempResponse{}, errors.New("bigfail")},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s: %s %s", tc.name, tc.req.Method, tc.req.URL), func(t *testing.T) {
			// TODO: if this becomes anything besides DumbClient, it's time to change this assertion
			client := (*tc.client).(DumbClient)
			client.Err = tc.expectedErr
			got, err := relayRequest(tc.req, client)
			if tc.expectedErr != nil {
				if err == nil {
					t.Errorf("err: got %v, want %v", err, tc.expectedErr)
				}
			} else {
				if !reflect.DeepEqual(got.Body, tc.want.Body) {
					t.Errorf("request.Body: got %s, want %s", got.Body, tc.want.Body)
				}

				if got.StatusCode != tc.want.StatusCode {
					t.Errorf("request.Method: got %d, want %d", got.StatusCode, tc.want.StatusCode)
				}

				if !reflect.DeepEqual(got.Header, tc.want.Header) {
					t.Errorf("request.Body: got \n%s\nwant \n%s\n", prettyHeader(got.Header), prettyHeader(tc.want.Header))
				}
			}
		})
	}
}

func prettyHeader(header http.Header) (output string) {

	for name, values := range header {
		offset := len(name) + 4
		output += name + ": [\n"
		for _, value := range values {
			output += fmt.Sprintf("%s'%s'\n", strings.Repeat(" ", offset), strings.ReplaceAll(value, " ", "…"))
		}
		output += "]\n"
	}

	return output
}

type genUpstreamRequest struct {
	req         *http.Request
	want        *http.Request
	expectedErr error
}

func TestGenerateUpstreamRequest(t *testing.T) {

	stdHeader := http.Header{
		"Key1": {"1a", "1b", "1c"},
		"Key2": {"2a", "2b", "2c"},
		"Key3": {"3a", "3b", "3c"},
	}
	stdBody := io.NopCloser(strings.NewReader("Woohoo!\noh no\n WOOHOO! "))
	r, _ := http.NewRequest("GET", "http://google.com/a/b/c", stdBody)

	cases := []genUpstreamRequest{
		{
			&http.Request{Host: "google.com",
				URL:    &url.URL{Path: "/a/b/c"},
				Method: "GET",
				Body:   stdBody,
				Header: stdHeader},
			&http.Request{URL: r.URL,
				Method: "GET",
				Body:   stdBody,
				Header: stdHeader},
			nil,
		},
		{
			&http.Request{Host: "		",
				URL:    &url.URL{Path: ""},
				Method: "GUT",
				Body:   stdBody,
				Header: stdHeader},
			nil,
			errors.New("?"),
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s %s%s", tc.req.Method, tc.req.Host, tc.req.URL.String()), func(t *testing.T) {
			got, err := generateUpstreamRequest(tc.req)

			if tc.expectedErr != nil {
				if err == nil {
					t.Errorf("err: got %v, want %v", err, tc.expectedErr)
				}
				if got != nil {
					t.Error("req: request non-nil on error")
				}

			} else {
				if got.URL.String() != tc.want.URL.String() {
					t.Errorf("request.URL: got %s, want %s", got.URL.String(), tc.want.URL.String())
				}

				if got.Body != tc.want.Body {
					t.Errorf("request.Body: got %s, want %s", got.Body, tc.want.Body)
				}

				if got.Method != tc.want.Method {
					t.Errorf("request.Method: got %s, want %s", got.Method, tc.want.Method)
				}

				if !reflect.DeepEqual(got.Header, tc.want.Header) {
					t.Errorf("request.Body: got \n%s\nwant \n%s\n", prettyHeader(got.Header), prettyHeader(tc.want.Header))
				}
			}
		})
	}
}

// type formUpstreamResponse struct {
// 	res         tempResponse
// 	want        *tempResponse
// 	expectedErr error
// }

// func TestGenerateUpstreamRequest(t *testing.T) {

// 	stdHeader := http.Header{
// 		"Key1": {"1a", "1b", "1c"},
// 		"Key2": {"2a", "2b", "2c"},
// 		"Key3": {"3a", "3b", "3c"},
// 	}
// 	stdBody := io.NopCloser(strings.NewReader("Woohoo!\noh no\n WOOHOO! "))
// 	r, _ := http.NewRequest("GET", "http://google.com/a/b/c", stdBody)

// 	cases := []formUpstreamResponse{
// 		{
// 			&http.Response{StatusCode: 200,
// 				Body:   stdBody,
// 				Header: stdHeader},
// 			&tempResponse{StatusCode: 200,
// 				Body:   stdBody,
// 				Header: stdHeader},
// 			nil,
// 		}
// 	}

// 	for _, tc := range cases {
// 		t.Run(fmt.Sprintf("%s %s%s", tc.res., tc.res.Host, tc.res.URL.String()), func(t *testing.T) {
// 			got, err := formatUpstreamResponse(tc.res)

// 			if tc.expectedErr != nil {
// 				if err == nil {
// 					t.Errorf("err: got %v, want %v", err, tc.expectedErr)
// 				}
// 				if got != nil {
// 					t.Error("req: request non-nil on error")
// 				}

// 			} else {
// 				if got.Body != tc.want.Body {
// 					t.Errorf("request.Body: got %s, want %s", got.Body, tc.want.Body)
// 				}

// 				if got.StatusCode != tc.want.StatusCode {
// 					t.Errorf("request.StatusCode: got %s, want %s", got.StatusCode, tc.want.StatusCode)
// 				}

// 				if !reflect.DeepEqual(got.Header, tc.want.Header) {
// 					t.Errorf("request.Body: got \n%s\nwant \n%s\n", prettyHeader(got.Header), prettyHeader(tc.want.Header))
// 				}
// 			}
// 		})
// 	}
// }
