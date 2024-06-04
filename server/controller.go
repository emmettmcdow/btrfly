package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	// "github.com/emmettmcdow/kache/server/proxy"
)

func main2() {
	m := http.NewServeMux()
	s := http.Server{Addr: ":81", Handler: m}
	m.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Shutting down", 400)
		s.Shutdown(context.Background())
	})
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		id, ok := r.Header["ID"]
		if !ok {
			http.Error(w,
				"No 'ID' header was passed",
				http.StatusBadRequest)
		}
		err := Login(id[0])
		if err != nil {
			http.Error(w,
				fmt.Sprintf("Invalid ID: %s", err),
				http.StatusBadRequest)
		}
	})
	m.HandleFunc("/tag", func(w http.ResponseWriter, r *http.Request) {
		tag, ok := r.Header["Tag"]
		if !ok {
			http.Error(w,
				"No 'Tag' header was passed",
				http.StatusBadRequest)
		}
		Tag(tag[0])
	})
	m.HandleFunc("/mode", func(w http.ResponseWriter, r *http.Request) {
		mode, ok := r.Header["Mode"]
		if !ok {
			http.Error(w,
				"No 'Mode' header was passed",
				http.StatusBadRequest)
		}
		err := Mode(mode[0])
		if err != nil {
			http.Error(w,
				fmt.Sprintf("Failed to change mode: %s", err),
				http.StatusBadRequest)
		}
	})
	log.Print(s.ListenAndServe())
}
