package main

import "os/exec"
// import "log"
import "fmt"
import "strings"
import "os"
import "flag"

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
		MacDNSConfig(*configIpPtr)
	case "deconfig":
		deconfigMode.Parse(os.Args[1:])
		MacDNSDeconfig()
		MacDNSFlushCache()
	default:
		fmt.Println("Expected either 'config' or 'deconfig'")
		os.Exit(1)
	}
}

func MacDNSConfig(ip string) {
	var interfaces = MacGetAllNetworkServices()

	for _, interface_name := range interfaces {
		MacDNSConfigForInterface(interface_name, ip)
	}
}

func MacDNSDeconfig() {
	var interfaces = MacGetAllNetworkServices()

	for _, interface_name := range interfaces {
		MacDNSDeconfigForInterface(interface_name)
	}
}

func MacDNSConfigForInterface(interface_name string, ip string) {
	var cmd = exec.Command("networksetup", "-setdnsservers", interface_name, ip)
	var out strings.Builder
	var stderr strings.Builder

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
	    fmt.Println(fmt.Sprint(err) + "2: " + stderr.String())
		return
	}
}

func DeleteEmpty (s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func MacGetAllNetworkServices() (interfaces []string) {
	var cmd = exec.Command("networksetup", "-listallnetworkservices")
	var out strings.Builder
	var stderr strings.Builder

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
	    fmt.Println(fmt.Sprint(err) + "3: " + stderr.String())
	}

	interfaces = DeleteEmpty(strings.Split(out.String(), "\n")[1:])

	return interfaces
}

func MacDNSFlushCache() {
	var cmd = exec.Command("dscacheutil", "-flushcache")
	var out strings.Builder
	var stderr strings.Builder

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
	    fmt.Println(fmt.Sprint(err) + "4: " + stderr.String())
	    return
	}
	// TODO: do something about this sudo? Seems the only one that requires it...
	cmd = exec.Command("sudo", "killall", "-HUP", "mDNSResponder")
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
	    fmt.Println(fmt.Sprint(err) + "5: " + stderr.String())
	    return
	}
}

func MacDNSDeconfigForInterface(interface_name string) {
	var cmd = exec.Command("networksetup", "-setdnsservers", interface_name, "Empty")
	var out strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
	    fmt.Println(fmt.Sprint(err) + "6: " + stderr.String())
		return
	}
}
