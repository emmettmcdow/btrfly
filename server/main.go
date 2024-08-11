package main

import (
	"context"
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
		defer proxyServer.Shutdown(context.TODO())
		defer proxyServerTLS.Shutdown(context.TODO())
		defer controllerServer.Shutdown(context.TODO())
		for sig := range c {
			if sig == syscall.SIGINT {
				log.Println("Recieved keyboard interrupt. Shutting down server.")
				break
			}
		}
	}()

	wg.Wait()
}
