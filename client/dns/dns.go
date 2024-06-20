package dns

import (
	"flag"
	"fmt"
	"os"
)

// TODO: add some setting that controls whether or not sudo is run

func main() {
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
		Config(*configIpPtr)
	case "deconfig":
		deconfigMode.Parse(os.Args[1:])
		Deconfig()
		FlushCache()
	default:
		fmt.Println("Expected either 'config' or 'deconfig'")
		os.Exit(1)
	}
}
