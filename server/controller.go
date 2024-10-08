package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
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

	address := fmt.Sprintf(":%d", port)
	s = &http.Server{Addr: address, Handler: m, TLSConfig: config}
	// TODO: add login back
	// m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
	// 	id, ok := r.Header["Id"]
	// 	if !ok {
	// 		http.Error(w,
	// 			"No 'ID' header was passed",
	// 			http.StatusBadRequest)
	// 		return
	// 	}
	// 	err := Login(id[0])
	// 	if err != nil {
	// 		http.Error(w,
	// 			fmt.Sprintf("Invalid ID: %s", err),
	// 			http.StatusBadRequest)
	// 		return
	// 	}
	// })
	m.HandleFunc("/tag", func(w http.ResponseWriter, r *http.Request) {
		tag, ok := r.Header["Tag"]
		if !ok {
			http.Error(w,
				"No 'Tag' header was passed",
				http.StatusBadRequest)
			return
		}
		if err := Tag(tag[0]); err != nil {
			fmt.Printf("Failed to Tag %s: %s\n", tag[0], err)
		}
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
	m.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("healthy"))
		if err != nil {
			fmt.Printf("Failed to write response: %s", err)
		}
	})
	go func() {
		defer wg.Done()
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe failed: %s\n", err)
		}
	}()

	// Block until server is ready
	healthClient := http.DefaultClient
	for i := 0; i < 5; i += 1 {
		res, err := healthClient.Get("http://" + "127.0.0.1" + address + "/health")
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if res.StatusCode == 200 {
			return s
		}
		time.Sleep(5 * time.Second)
	}

	fmt.Printf("Health check failed")
	return nil
}
