package main

import (
	"flag"
	"fmt"
	"github.com/emmettmcdow/btrfly/client/dns"
	"io"
	"net/http"
	"os"
)

var client *http.Client

const defaultCtrlEndpoint string = "127.0.0.1:5678"
const defaultDNSEndpoint string = "127.0.0.1"

type defaultDns struct{}

func (d defaultDns) Config(ip string) (err error) {
	return dns.Config(ip)
}
func (d defaultDns) Deconfig() (err error) {
	return dns.Deconfig()
}
func (d defaultDns) FlushCache() (err error) {
	return dns.FlushCache()
}

type DNSConfig interface {
	Config(ip string) (err error)
	Deconfig() (err error)
	FlushCache() (err error)
}

func main() {
	flag.Parse()
	args := flag.Args()
	arglen := len(flag.Args())
	out := _main(defaultDns{}, defaultCtrlEndpoint, arglen, args)
	os.Exit(out)
}

// TODO: add login back
// TODO: pass config and deconfig errors back up!
func _main(dns DNSConfig, ctrlEndpoint string, arglen int, args []string) int {
	var subcommand string
	client = &http.Client{}

	if arglen == 0 {
		subcommand = "help"
	} else {
		subcommand = args[0]
	}
	switch subcommand {
	case "config":
		var dnsEndpoint string
		if arglen == 2 {
			dnsEndpoint = args[1]
		} else if arglen == 1 {
			dnsEndpoint = defaultDNSEndpoint
		} else {
			fmt.Fprintf(os.Stderr, "Unrecognized arguments.\n")
			return 1
		}
		err := dns.Config(dnsEndpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to configure DNS: %s\n", err)
		}
		err = dns.FlushCache()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to flush DNS cache: %s\n", err)
		}

	case "deconfig":
		if arglen != 1 {
			fmt.Fprintf(os.Stderr, "Unrecognized arguments.\n")
			return 1
		}
		err := dns.Deconfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to deconfigure DNS: %s\n", err)
		}
		err = dns.FlushCache()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to flush DNS cache: %s\n", err)
		}
	case "tag":
		if arglen != 2 {
			fmt.Fprintf(os.Stderr, "No tag given.\n")
			return 1
		}
		if err := tag(args[1], ctrlEndpoint); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set the tag: %s\n\n", err)
			return 1
		}
	// case "login":
	// 	if arglen != 2 {
	// 		fmt.Fprintf(os.Stderr, "No login id given.\n")
	// 		return 1
	// 	}
	// 	if err := login(args[1], ctrlEndpoint); err != nil {
	// 		fmt.Fprintf(os.Stderr, "Failed to login: %s\n\n", err)
	// 		return 1
	// 	}
	case "mode":
		if arglen != 2 {
			fmt.Fprintf(os.Stderr, "No mode given.\n")
			return 1
		}
		if err := mode(args[1], ctrlEndpoint); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set mode: %s\n\n", err)
			return 1
		}
	case "help":
		if arglen == 2 {
			switch args[1] {
			case "config":
				fmt.Printf("Help: btrfly config [dns_server]\n")
				fmt.Printf("    config - configure this machine to utilize the btrfly server\n")
				fmt.Printf("    Takes an optional argument [dns server]. This overrides the default dns\n")
				fmt.Printf("    server.\n")
			case "deconfig":
				fmt.Printf("Help: btrfly deconfig\n")
				fmt.Printf("    deconfigure - unsets the dns server set by config.\n")
			case "tag":
				fmt.Printf("Help: btrfly tag tag_name\n")
				fmt.Printf("    tag - set the tag to identify this current build\n")
				fmt.Printf("    tag_name is required and passed as an argument.\n")
			case "mode":
				fmt.Printf("Help: btrfly mode mode_ver\n")
				fmt.Printf("    mode - change the mode of operation of the btrfly service\n")
				fmt.Printf("    mode_verb is required and passed as an argument.\n")
				fmt.Printf("    mode_verb is one of: record, playback, standby.\n")
			// case "login":
			// 	fmt.Printf("Help: btrfly login id\n")
			// 	fmt.Printf("    login - set your credentials so that you can use the btrfly service.\n")
			// 	fmt.Printf("    id is required and passed as an argument.\n")
			case "help":
				defaultHelp()
			default:
				fmt.Printf("%s is not a valid subcommand\n", args[1])
				defaultHelp()
			}
		} else {
			defaultHelp()
			return 0
		}
	default:
		fmt.Printf("%s is not a valid subcommand\n", subcommand)
		defaultHelp()
		return 1
	}
	return 0
}

func defaultHelp() {
	fmt.Printf("btrfly Client CLI\n")
	fmt.Printf("Available subcommands:\n")
	fmt.Printf("    config   - configure this machine to utilize the btrfly server\n")
	fmt.Printf("    deconfig - deconfigure this machine (...)\n")
	fmt.Printf("    tag      - set the tag to identify this current build\n")
	// fmt.Printf("    login    - set your credentials so that you can use the btrfly service\n")
	fmt.Printf("    mode     - change the mode of operation of the btrfly service\n")
	fmt.Printf("    help     - pass another subcommand to get info about that subcommand\n")
}

func tag(tag string, ctrlEndpoint string) (err error) {
	req, err := http.NewRequest("GET", "http://"+ctrlEndpoint+"/tag", http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Add("Tag", tag)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform http request: %s", err)
	}
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("got response code %d, and failed to read body", resp.StatusCode)
		}
		return fmt.Errorf("got response code %d with body:\n%s", resp.StatusCode, body)
	}
	return nil
}

// func login(id string, ctrlEndpoint string) (err error) {
// 	req, err := http.NewRequest("GET", "http://"+ctrlEndpoint+"/login", http.NoBody)
// 	if err != nil {
// 		return err
// 	}
// 	req.Header.Add("Id", id)
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("failed to perform http request: %s", err)
// 	}
// 	if resp.StatusCode != 200 {
// 		body, err := io.ReadAll(resp.Body)
// 		if err != nil {
// 			return fmt.Errorf("got response code %d, and failed to read body", resp.StatusCode)
// 		}
// 		return fmt.Errorf("got response code %d with body:\n%s", resp.StatusCode, body)
// 	}
// 	return nil
// }

func mode(mode string, ctrlEndpoint string) (err error) {
	var modeI string
	switch mode {
	case "record":
		modeI = "0"
	case "playback":
		modeI = "1"
	case "standby":
		modeI = "2"
	default:
		return fmt.Errorf("mode '%s' is not a valid mode", mode)
	}
	req, err := http.NewRequest("GET", "http://"+ctrlEndpoint+"/mode", http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Add("Mode", modeI)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform http request: %s", err)
	}
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("got response code %d, and failed to read body", resp.StatusCode)
		}
		return fmt.Errorf("got response code %d with body:\n%s", resp.StatusCode, body)
	}
	return nil
}
