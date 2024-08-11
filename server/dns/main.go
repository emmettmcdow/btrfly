package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const DEFAULT_ADDR = "127.0.0.1:53"

var DEFAULT_RESPONSE = []byte{127, 0, 0, 1}

func dns(listen string) {
	// Resolve the string address to a UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", listen)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Start listening for UDP packages on the given address
	conn, err := net.ListenUDP("udp", udpAddr)
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Printf("Failed to close UDP connection: %s\n", err)
		}
	}()
	fmt.Printf("DNS server listening on %s\n", DEFAULT_ADDR)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
		var buf [512]byte
		_, client, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			fmt.Println(err)
			return
		}
		response, err := Deserialize(buf[0:])
		if err != nil {
			fmt.Printf("Failed to deserialize message: %s", err)
			continue
		}
		response.header.ancount = 1
		response.header.qdcount = 1
		response.header.nscount = 0
		response.header.artcount = 0
		_, opcode, aa, tc, rd, ra, z, rcode := unpacked(response.header.packed)
		qr := uint8(1)
		ra = 1
		response.header.packed = packed(qr, opcode, aa, tc, rd, ra, z, rcode)

		answer := Answer{
			name:     response.questions[0].qname,
			kind:     response.questions[0].qtype,
			class:    response.questions[0].qclass,
			ttl:      420,
			rdlength: 4,
			rdata:    DEFAULT_RESPONSE,
		}
		response.answers = append(response.answers, answer)
		data, err := Serialize(response)
		if err != nil {
			fmt.Printf("Failed to serialize message: %s", err)
			continue
		}
		_, err = conn.WriteToUDP(data, client)
		if err != nil {
			fmt.Printf("Failed to write to UDP: %s\n", err)
		}
	}
}

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	dns(DEFAULT_ADDR)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig == syscall.SIGINT {
				log.Println("Recieved keyboard interrupt. Shutting down server.")
				break
			}
		}
	}()

	wg.Wait()
}
