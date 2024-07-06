package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	// "github.com/emmettmcdow/kache/server/proxy"
)

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s := handler(wg, 5678)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		// TODO: figure out this TODO business with the context
		defer s.Shutdown(context.TODO())
		for sig := range c {
			if sig == syscall.SIGINT {
				log.Println("Recieved keyboard interrupt. Shutting down server.")
				break
			}
		}
	}()

	wg.Wait()
}

// TODO: make the controller API actually not shit(comply with REST)
func handler(wg *sync.WaitGroup, port uint) (s *http.Server) {
	m := http.NewServeMux()
	s = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: m}
	// TODO: remove this ASAP
	m.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Shutting down", 400)
		s.Shutdown(context.Background())
	})
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
