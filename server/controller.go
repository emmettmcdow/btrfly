package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	// "github.com/emmettmcdow/btrfly/server/proxy"
)

// TODO: make the controller API actually not shit(comply with REST)
func controller(wg *sync.WaitGroup, port uint, tlsEnabled bool) (s *http.Server) {
	var config *tls.Config

	m := http.NewServeMux()
	if tlsEnabled {
		cert, err := tls.LoadX509KeyPair("server.pem", "server.key")
		if err != nil {
			fmt.Printf("Failed to load certificate keypair: %s\n", err)
		}
		config = &tls.Config{Certificates: []tls.Certificate{cert}}
	}
	s = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: m, TLSConfig: config}
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		id, ok := r.Header["Id"]
		if !ok {
			http.Error(w,
				"No 'ID' header was passed",
				http.StatusBadRequest)
			return
		}
		err := Login(id[0])
		if err != nil {
			http.Error(w,
				fmt.Sprintf("Invalid ID: %s", err),
				http.StatusBadRequest)
			return
		}
	})
	m.HandleFunc("/tag", func(w http.ResponseWriter, r *http.Request) {
		tag, ok := r.Header["Tag"]
		if !ok {
			http.Error(w,
				"No 'Tag' header was passed",
				http.StatusBadRequest)
			return
		}
		Tag(tag[0])
	})
	m.HandleFunc("/mode", func(w http.ResponseWriter, r *http.Request) {
		mode, ok := r.Header["Mode"]
		if !ok {
			http.Error(w,
				"No 'Mode' header was passed",
				http.StatusBadRequest)
			return
		}
		err := Mode(mode[0])
		if err != nil {
			http.Error(w,
				fmt.Sprintf("Failed to change mode: %s", err),
				http.StatusBadRequest)
			return
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
