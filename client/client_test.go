package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func fakeController(wg *sync.WaitGroup, talkback chan<- string, port int) (s *http.Server) {

	m := http.NewServeMux()
	s = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: m}
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		id, ok := r.Header["Id"]
		headers := "Headers: ["
		if ok {
			headers += fmt.Sprintf("ID: '%s',", id)
		}
		tag, ok := r.Header["Tag"]
		if ok {
			headers += fmt.Sprintf("Tag: '%s',", tag)
		}
		mode, ok := r.Header["Mode"]
		if ok {
			headers += fmt.Sprintf("Mode: '%s',", mode)
		}
		headers += "]"
		method := r.Method
		path := r.URL.String()

		talkback <- fmt.Sprintf("<%s> %s - %s", method, path, headers)
	})
	go func() {
		defer wg.Done()
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe failed: %s\n", err)
		}
	}()

	return s
}

type FakeConfig struct {
	ConfigCalled   int
	CtrlEndpoint   string
	DeconfigCalled int
	FlushCalled    int
}

func (f *FakeConfig) Config(ip string) (err error) {
	if ip != "" {
		f.CtrlEndpoint = ip
	}
	f.ConfigCalled += 1
	return nil
}
func (f *FakeConfig) Deconfig() (err error) {
	f.DeconfigCalled += 1
	return nil
}
func (f *FakeConfig) FlushCache() (err error) {
	f.FlushCalled += 1
	return nil
}

func consumeRequest(talkback <-chan string) (output string) {
	select {
	case output = <-talkback:
		return output
	case <-time.After(3 * time.Second):
		return ""
	}
}

func TestClient(t *testing.T) {
	port := 8081
	talkback := make(chan string, 1)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	controllerServer := fakeController(wg, talkback, port)
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer func() {
		err := controllerServer.Shutdown(timeout)
		if err != nil {
			fmt.Printf("failed to shutdown controllerServer: %s", err)
		}
	}()

	subtests := []struct {
		command   []string
		errno     int
		req       string
		nconfigs  int
		ndconfigs int
		nflushes  int
	}{
		{[]string{"config"}, 0, "", 1, 0, 1},
		{[]string{"config", "127.0.0.1:420"}, 0, "", 1, 0, 1},
		{[]string{"config", "127.0.0.1:420", "127.0.0.1:422"}, 1, "", 0, 0, 0},
		{[]string{"deconfig"}, 0, "", 0, 1, 1},
		{[]string{"deconfig", "127.0.0.1:420"}, 1, "", 0, 0, 0},
		{[]string{"tag", "tag-working"}, 0, "<GET> /tag - Headers: [Tag: '[tag-working]',]", 0, 0, 0},
		{[]string{"tag", "very_very_long_and_complicated-tag-with-68-numbers_and-mixed_"}, 0, "<GET> /tag - Headers: [Tag: '[very_very_long_and_complicated-tag-with-68-numbers_and-mixed_]',]", 0, 0, 0},
		{[]string{"tag"}, 1, "", 0, 0, 0},
		{[]string{"mode", "record"}, 0, "<GET> /mode - Headers: [Mode: '[0]',]", 0, 0, 0},
		{[]string{"mode", "playback"}, 0, "<GET> /mode - Headers: [Mode: '[1]',]", 0, 0, 0},
		{[]string{"mode", "standby"}, 0, "<GET> /mode - Headers: [Mode: '[2]',]", 0, 0, 0},
		{[]string{"mode"}, 1, "", 0, 0, 0},
		{[]string{"mode", "standby", "uhoh"}, 1, "", 0, 0, 0},
		{[]string{"login", "420"}, 0, "<GET> /login - Headers: [ID: '[420]',]", 0, 0, 0},
		{[]string{"login", "690000"}, 0, "<GET> /login - Headers: [ID: '[690000]',]", 0, 0, 0},
		{[]string{"login", "abc"}, 1, "", 0, 0, 0},
		{[]string{"login", "1", "1"}, 1, "", 0, 0, 0},
		{[]string{"login"}, 1, "", 0, 0, 0},
		{[]string{}, 1, "", 0, 0, 0},
		{[]string{"gobbledygook"}, 1, "", 0, 0, 0},
		{[]string{"help"}, 0, "", 0, 0, 0},
		{[]string{"help", "config"}, 0, "", 0, 0, 0},
		{[]string{"help", "deconfig"}, 0, "", 0, 0, 0},
		{[]string{"help", "tag"}, 0, "", 0, 0, 0},
		{[]string{"help", "mode"}, 0, "", 0, 0, 0},
		{[]string{"help", "login"}, 0, "", 0, 0, 0},
		{[]string{"help", "gobbledygook"}, 0, "", 0, 0, 0},
		{[]string{"help", "gobbledygook", "g2"}, 0, "", 0, 0, 0},
	}

	for _, st := range subtests {
		t.Run(strings.Join(st.command, " "), func(t *testing.T) {
			fakeConfig := &FakeConfig{}
			_main(fakeConfig, fmt.Sprintf("127.0.0.1:%d", port), len(st.command), st.command)
			if fakeConfig.ConfigCalled != st.nconfigs {
				t.Errorf("Expected %d call(s) to Config, got %d\n", st.nconfigs, fakeConfig.ConfigCalled)
			}
			if fakeConfig.DeconfigCalled != st.ndconfigs {
				t.Errorf("Expected %d call(s) to Deconfig, got %d\n", st.ndconfigs, fakeConfig.DeconfigCalled)
			}
			if fakeConfig.FlushCalled != st.nflushes {
				t.Errorf("Expected %d call(s) to Flush, got %d\n", st.nflushes, fakeConfig.FlushCalled)
			}
			if st.req != "" {
				gotReq := consumeRequest(talkback)
				if st.req != gotReq {
					t.Errorf("Expected this to be sent: %s, got: %s\n", st.req, gotReq)
				}
			}
		})
	}
}
