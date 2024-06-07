package main

import (
	"net/http"
	"net/url"
	"os"
	"fmt"
	"io"
	"flag"
	"github.com/emmettmcdow/kache/client/dns"
)

var client *http.Client
var d dns.DNSConfig
var ctrlEndpoint string
var proxyEndpoint string

const	defaultCtrlEndpoint string = "127.0.0.1:81"
	// defaultProxyEndpoint := "127.0.0.1:80"
const	defaultDNSEndpoint string =  "127.0.0.1:53"
func main() {
	client = &http.Client{}
	d = dns.ConfigAgent()


	// configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	
	// deconfigCmd := flag.NewFlagSet("deconfig", flag.ExitOnError)

	// modeCmd := flag.NewFlagSet("mode", flag.ExitOnError)

	// loginCmd := flag.NewFlagSet("login", flag.ExitOnError)
	
	// tagCmd := flag.NewFlagSet("tag", flag.ExitOnError)

	flag.Parse()
	arglen := len(flag.Args())
	switch os.Args[1] {
	case "config":
		var dnsEndpoint string
		if arglen == 1 {
			dnsEndpoint = flag.Args()[0]
		} else if arglen == 0 {
			dnsEndpoint = defaultDNSEndpoint
		} else {
			fmt.Fprintf(os.Stderr, "Unrecognized arguments.\n")
			os.Exit(1)
		}
		d.Config(dnsEndpoint)
		d.FlushCache()
	case "deconfig":
		if arglen != 0 {
			fmt.Fprintf(os.Stderr, "Unrecognized arguments.\n")
			os.Exit(1)
		}
		d.Deconfig()
		d.FlushCache()
	case "tag":
		if arglen != 1 {
			fmt.Fprintf(os.Stderr, "No tag given.\n")
			os.Exit(1)
		}
		if err := tag(flag.Args()[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set the tag: %s\n\n", err)
			os.Exit(1)
		}
	case "login":
		if arglen != 1 {
			fmt.Fprintf(os.Stderr, "No login id given.\n")
			os.Exit(1)
		}
		if err := login(flag.Args()[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to login: %s\n\n", err)
			os.Exit(1)
		}
	case "mode":
		if arglen != 1 {
			fmt.Fprintf(os.Stderr, "No mode given.\n")
			os.Exit(1)
		}
		if err := mode(flag.Args()[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set mode: %s\n\n", err)
			os.Exit(1)
		}
	}
}

func tag(tag string) (err error) {
	req := &http.Request{Host: defaultCtrlEndpoint,
						URL:    &url.URL{Path: "/tag"},
						Method: "GET",
						Header: http.Header{"Tag": []string{tag}}}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to perform http request: %s", err)
	}
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Got response code %d, and failed to read body", resp.StatusCode)
		}
		return fmt.Errorf("Got response code %d with body:\n%s\n", resp.StatusCode, body)
	}
	return nil
}

func login(id string) (err error) {
	req := &http.Request{Host: defaultCtrlEndpoint,
						URL:    &url.URL{Path: "/login"},
						Method: "GET",
						Header: http.Header{"ID": []string{id}}}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to perform http request: %s", err)
	}
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Got response code %d, and failed to read body", resp.StatusCode)
		}
		return fmt.Errorf("Got response code %d with body:\n%s\n", resp.StatusCode, body)
	}
	return nil
}

func mode(mode string) (err error) {
	var modeI string
	switch mode {
	case "record":
		modeI = "0"
	case "playback":
		modeI = "1"
	case "standby":
		modeI = "2"
	default:
		return fmt.Errorf("Mode '%s' is not a valid mode.", mode)
	}
	req := &http.Request{Host: defaultCtrlEndpoint,
						URL:    &url.URL{Path: "/mode"},
						Method: "GET",
						Header: http.Header{"Mode": []string{modeI}}}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to perform http request: %s", err)
	}
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Got response code %d, and failed to read body", resp.StatusCode)
		}
		return fmt.Errorf("Got response code %d with body:\n%s\n", resp.StatusCode, body)
	}
	return nil
}
