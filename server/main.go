package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	controllerServer := controller(wg, 5678, true)
	wg.Add(1)
	proxyServer := proxy(wg, 80, false)
	wg.Add(1)
	proxyServerTLS := proxy(wg, 443, true)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		defer func() {
			err := proxyServer.Shutdown(timeout)
			if err != nil {
				fmt.Printf("failed to shutdown proxyServer: %s", err)
			}
		}()
		defer func() {
			err := proxyServerTLS.Shutdown(timeout)
			if err != nil {
				fmt.Printf("failed to shutdown proxyServerTLS: %s", err)
			}
		}()
		defer func() {
			err := controllerServer.Shutdown(timeout)
			if err != nil {
				fmt.Printf("failed to shutdown controllerServer: %s", err)
			}
		}()
		for sig := range c {
			if sig == syscall.SIGINT {
				log.Println("Recieved keyboard interrupt. Shutting down server.")
			}
		}
	}()

	wg.Wait()
}
