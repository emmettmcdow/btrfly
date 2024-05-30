package main

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"io"
	"bytes"
	"errors"
)

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

type relayRequestCase struct {
	name string
	req *http.Request
	client *clientSender
	want tempResponse
	expectedErr error
}

func TestRelayRequest(t *testing.T) {
	var fakeClient clientSender
	
	fmt.Println("Begin")
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
	io.Copy(&successResponse.Body, stdBody)
	stdBody.Reset(stdText)

	fakeClient = DumbClient{Err: nil}
	
	cases := []relayRequestCase {
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
					t.Errorf("request.Body: got %s, want %s", got.Body.String(), tc.want.Body.String())
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
		output += fmt.Sprintf("%s: [\n", name)
		for _, value := range values {
			output += fmt.Sprintf("%s'%s'\n", strings.Repeat(" ", offset), strings.ReplaceAll(value, " ", "…"))
		}
		output += fmt.Sprint("]\n")
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
