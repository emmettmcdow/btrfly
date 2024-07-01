package dns

import (
	"fmt"
	"os/exec"
	"strings"
)

func Config(ip string) (err error) {
	var interfaces = macGetAllNetworkServices()

	for _, interface_name := range interfaces {
		if err = macDNSConfigForInterface(interface_name, ip); err != nil {
			return err
		}
	}
	return nil
}

func Deconfig() (err error) {
	var interfaces = macGetAllNetworkServices()

	for _, interface_name := range interfaces {
		if err = macDNSDeconfigForInterface(interface_name); err != nil {
			return err
		}
	}
	return nil
}

func FlushCache() (err error) {
	var cmd = exec.Command("dscacheutil", "-flushcache")
	var out strings.Builder
	var stderr strings.Builder

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run [dscacheutil...]. Stderr: %s", stderr.String())
	}
	// TODO: do something about this sudo? Seems the only one that requires it...
	cmd = exec.Command("sudo", "killall", "-HUP", "mDNSResponder")
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run [killall mDNSResponsder...]. Stderr: %s", stderr.String())
	}
	return nil
}
func macDNSConfigForInterface(interface_name string, ip string) (err error) {
	var cmd = exec.Command("networksetup", "-setdnsservers", interface_name, ip)
	var out strings.Builder
	var stderr strings.Builder

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to run [networksetup -setdnsservers...]. Stderr: %s", stderr.String())
	}
	return nil
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func macGetAllNetworkServices() (interfaces []string) {
	var cmd = exec.Command("networksetup", "-listallnetworkservices")
	var out strings.Builder
	var stderr strings.Builder

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Failed to run [networksetup -listallnetworkservices...]. Stderr: %s\n", stderr.String())
	}

	interfaces = deleteEmpty(strings.Split(out.String(), "\n")[1:])

	return interfaces
}

func macDNSDeconfigForInterface(interface_name string) (err error) {
	var cmd = exec.Command("networksetup", "-setdnsservers", interface_name, "Empty")
	var out strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		fmt.Errorf("Failed to run [networksetup -setdnsservers...]. Stderr: %s", stderr.String())
	}
	return nil
}
