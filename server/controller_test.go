package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
)

type state struct {
	mode ProxyMode
	tag  string
	user uint64
}

// Base state
var baseWant = state{
	mode: MODE_S,
	tag:  "shoop da woop",
	user: 0,
}

func verifyState(wantState state, t *testing.T) {
	if proxyMode != wantState.mode {
		t.Errorf("proxyMode: Got: %s, Want: %s\n", proxyMode.String(), wantState.mode.String())
	}
	if buildTag != wantState.tag {
		t.Errorf("buildTag: Got: %s, Want: %s\n", buildTag, wantState.tag)
	}
	if currUser != wantState.user {
		t.Errorf("currUser: Got: %d, Want: %d\n", currUser, wantState.user)
	}
}

func testControllerMode(t *testing.T) {
	subtests := []struct {
		mode    uint8
		resCode int
	}{
		{uint8(MODE_P), 200},
		{uint8(MODE_R), 200},
		{244, 400},
		{69, 400},
	}
	client := &http.Client{}

	for _, st := range subtests {
		t.Run(fmt.Sprintf("MODE{%d}-GET{%d}", st.mode, st.resCode), func(t *testing.T) {
			want := baseWant
			verifyState(want, t)

			URL := "http://127.0.0.1:5678/mode"
			method := "GET"
			req, err := http.NewRequest(method, URL, http.NoBody)
			if err != nil {
				t.Errorf("Failed to generate new request for %s\n", URL)
			}
			req.Header.Set("Mode", strconv.FormatUint(uint64(st.mode), 10))
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Failed to \"Do\": %s\n", URL)
			}
			if resp.StatusCode != st.resCode {
				t.Errorf("Switching to mode %d: Got: %d, Want: %d\n", st.mode, resp.StatusCode, st.resCode)
			}
			if st.resCode == 200 {
				want.mode = ProxyMode(st.mode)
			}
			verifyState(want, t)

			// Reset to 0
			req, err = http.NewRequest(method, URL, http.NoBody)
			if err != nil {
				t.Errorf("Failed to generate new request for %s\n", URL)
			}
			req.Header.Set("Mode", "2")
			resp, err = client.Do(req)
			if err != nil {
				t.Errorf("Failed to \"Do\": %s\n", URL)
			}
			if resp.StatusCode != 200 {
				t.Errorf("Switching to mode %d: Got: %d, Want: %d\n", 0, resp.StatusCode, 200)
			}
			want = baseWant
			verifyState(want, t)
		})
	}
}

func testControllerTag(t *testing.T) {
	subtests := []struct {
		tag     string
		resCode int
	}{
		{"Elon Musk", 200},
		{"Buster Baxter from Arthur", 200},
		{"", 200}, // TODO: This should be 400 but we don't check anything
	}
	client := &http.Client{}

	for _, st := range subtests {
		t.Run(fmt.Sprintf("TAG{%s}-GET{%d}", st.tag, st.resCode), func(t *testing.T) {
			want := baseWant
			verifyState(want, t)

			URL := "http://127.0.0.1:5678/tag"
			method := "GET"
			req, err := http.NewRequest(method, URL, http.NoBody)
			if err != nil {
				t.Errorf("Failed to generate new request for %s\n", URL)
			}
			req.Header.Set("Tag", st.tag)
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Failed to \"Do\": %s\n", URL)
			}
			if resp.StatusCode != st.resCode {
				t.Errorf("Switching to tag %s: Got: %d, Want: %d\n", st.tag, resp.StatusCode, st.resCode)
			}
			if st.resCode == 200 {
				want.tag = st.tag
			}
			verifyState(want, t)

			// Reset to 0
			req, err = http.NewRequest(method, URL, http.NoBody)
			if err != nil {
				t.Errorf("Failed to generate new request for %s\n", URL)
			}
			req.Header.Set("Tag", "shoop da woop")
			resp, err = client.Do(req)
			if err != nil {
				t.Errorf("Failed to \"Do\": %s\n", URL)
			}
			if resp.StatusCode != 200 {
				t.Errorf("Switching to tag %s: Got: %d, Want: %d\n", "shoop da woop", resp.StatusCode, 200)
			}
			want = baseWant
			verifyState(want, t)
		})
	}
}

func testControllerLogin(t *testing.T) {
	subtests := []struct {
		id      string
		resCode int
	}{
		{"420", 200},
		{"4200", 200},
		{"-1", 400},
		{"", 400},
	}
	client := &http.Client{}

	for _, st := range subtests {
		t.Run(fmt.Sprintf("LOGIN{%s}-GET{%d}", st.id, st.resCode), func(t *testing.T) {
			want := baseWant
			verifyState(want, t)

			URL := "http://127.0.0.1:5678/login"
			method := "GET"
			req, err := http.NewRequest(method, URL, http.NoBody)
			if err != nil {
				t.Errorf("Failed to generate new request for %s\n", URL)
			}
			req.Header.Set("ID", st.id)
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Failed to \"Do\": %s\n", URL)
			}
			if resp.StatusCode != st.resCode {
				t.Errorf("Logging in as user %s: Got: %d, Want: %d\n", st.id, resp.StatusCode, 200)
			}
			if st.resCode == 200 {
				want.user, err = strconv.ParseUint(st.id, 10, 64)
				if err != nil {
					t.Errorf("Failed to convert st.id to uint: %s", err)
				}
			}
			verifyState(want, t)

			// Reset to 0
			req, err = http.NewRequest(method, URL, http.NoBody)
			if err != nil {
				t.Errorf("Failed to generate new request for %s\n", URL)
			}
			req.Header.Set("ID", "0")
			resp, err = client.Do(req)
			if err != nil {
				t.Errorf("Failed to \"Do\": %s\n", URL)
			}
			if resp.StatusCode != 200 {
				t.Errorf("Logging in as user 0: Got: %d, Want: %d\n", resp.StatusCode, 200)
			}
			want = baseWant
			verifyState(want, t)
		})
	}
}

func TestController(t *testing.T) {
	// Start up controller
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s := controller(wg, 5678, false)
	// TODO: ditch the TODO
	defer func() {
		err := s.Shutdown(context.TODO())
		if err != nil {
			t.Errorf("Failed to shutdown with error: %s\n", err)
		}
	}()

	subtests := []struct {
		name    string
		subtest func(t *testing.T)
	}{
		{"login", testControllerLogin},
		{"mode", testControllerMode},
		{"tag", testControllerTag},
	}

	for _, st := range subtests {
		t.Run(st.name, func(t *testing.T) { st.subtest(t) })
	}
}
