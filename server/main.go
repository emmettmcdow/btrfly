package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
		// TODO: figure out this TODO business with the context
		defer func() {
			err := controllerServer.Shutdown(context.TODO())
			if err != nil {
				fmt.Printf("Failed to shutdown with error: %s\n", err)
			}
		}()
		defer func() {
			err := proxyServer.Shutdown(context.TODO())
			if err != nil {
				fmt.Printf("Failed to shutdown with error: %s\n", err)
			}
		}()
		defer func() {
			err := proxyServerTLS.Shutdown(context.TODO())
			if err != nil {
				fmt.Printf("Failed to shutdown with error: %s\n", err)
			}
		}()
		for sig := range c {
			if sig == syscall.SIGINT {
				log.Println("Recieved keyboard interrupt. Shutting down server.")
				break
			}
		}
	}()

	wg.Wait()
}
