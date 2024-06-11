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
const	defaultDNSEndpoint string =  "127.0.0.1:53"

func main() {
	var subcommand string
	client = &http.Client{}
	d = dns.ConfigAgent()

	flag.Parse()
	arglen := len(flag.Args())
	if arglen == 0 {
		subcommand = "help"
	} else {
		subcommand = flag.Args()[0]
	}
	switch subcommand {
	case "config":
		var dnsEndpoint string
		if arglen == 2 {
			dnsEndpoint = flag.Args()[1]
		} else if arglen == 1 {
			dnsEndpoint = defaultDNSEndpoint
		} else {
			fmt.Fprintf(os.Stderr, "Unrecognized arguments.\n")
			os.Exit(1)
		}
		d.Config(dnsEndpoint)
		d.FlushCache()
	case "deconfig":
		if arglen != 1 {
			fmt.Fprintf(os.Stderr, "Unrecognized arguments.\n")
			os.Exit(1)
		}
		d.Deconfig()
		d.FlushCache()
	case "tag":
		if arglen != 2 {
			fmt.Fprintf(os.Stderr, "No tag given.\n")
			os.Exit(1)
		}
		if err := tag(flag.Args()[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set the tag: %s\n\n", err)
			os.Exit(1)
		}
	case "login":
		if arglen != 2 {
			fmt.Fprintf(os.Stderr, "No login id given.\n")
			os.Exit(1)
		}
		if err := login(flag.Args()[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to login: %s\n\n", err)
			os.Exit(1)
		}
	case "mode":
		if arglen != 2 {
			fmt.Fprintf(os.Stderr, "No mode given.\n")
			os.Exit(1)
		}
		if err := mode(flag.Args()[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set mode: %s\n\n", err)
			os.Exit(1)
		}
	case "help":
		if arglen == 2 {
			switch flag.Args()[1]{
			case "config":
				fmt.Printf("Help: kache config [dns_server]\n")
				fmt.Printf("    config - configure this machine to utilize the Kache server\n")
				fmt.Printf("    Takes an optional argument [dns server]. This overrides the default dns\n")
				fmt.Printf("    server\n")
			case "deconfigure":
				fmt.Printf("Help: kache deconfigure\n")
				fmt.Printf("    deconfigure - unsets the dns server set by config\n")
			case "tag":
				fmt.Printf("Help: kache tag tag_name")
				fmt.Printf("    tag - set the tag to identify this current build\n")
				fmt.Printf("    tag_name is required and passed as an argument\n")
			case "mode":
				fmt.Printf("Help: kache mode mode_ver")
				fmt.Printf("    mode - change the mode of operation of the kache service\n")
				fmt.Printf("    mode_verb is required and passed as an argument\n")
				fmt.Printf("    mode_verb is one of: record, playback, standby\n")
			case "login":
				fmt.Printf("Help: kache login id")
				fmt.Printf("    login - set your credentials so that you can use the Kache service\n")
				fmt.Printf("    id is required and passed as an argument\n")
			}
		} else {
			defaultHelp()
			os.Exit(0)
		}
	default:
		fmt.Printf("%s is not a valid subcommand", subcommand)
		defaultHelp()
	}
}

func defaultHelp() {
		fmt.Printf("Kache Client CLI\n")
		fmt.Printf("Available subcommands:\n")
		fmt.Printf("    config - configure this machine to utilize the Kache server\n")
		fmt.Printf("    deconfigure - deconfigure this machine (...)\n")
		fmt.Printf("    tag         - set the tag to identify this current build\n")
		fmt.Printf("    login       - set your credentials so that you can use the Kache service\n")
		fmt.Printf("    mode        - change the mode of operation of the kache service\n")
		fmt.Printf("    help        - pass another subcommand to get info about that subcommand\n")
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
