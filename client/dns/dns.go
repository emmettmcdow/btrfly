package dns

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

// TODO: add some setting that controls whether or not sudo is run

func main() {
	var dns DNSConfig

	configMode := flag.NewFlagSet("config", flag.ExitOnError)
	configIpPtr := flag.String("ip", "127.0.0.1", "Default DNS server to set")

	deconfigMode := flag.NewFlagSet("deconfig", flag.ExitOnError)

	if len(os.Args) == 1 {
		fmt.Println("Expected either 'config' or 'deconfig'")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "config":
		configMode.Parse(os.Args[1:])
		dns.Config(*configIpPtr)
	case "deconfig":
		deconfigMode.Parse(os.Args[1:])
		dns.Deconfig()
		dns.FlushCache()
	default:
		fmt.Println("Expected either 'config' or 'deconfig'")
		os.Exit(1)
	}
}

type DNSConfig interface {
	Config(string) (err error)
	Deconfig() (err error)
	FlushCache() (err error)
}

func ConfigAgent() (dns DNSConfig) {
	switch runtime.GOOS {
	case "linux":
		dns = LinuxDNS{}
	case "darwin":
		dns = MacDNS{}
	default:
		fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}
	return dns
}
