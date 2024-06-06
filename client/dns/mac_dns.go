package dns

import (
	"fmt"
	"os/exec"
	"strings"
)

type MacDNS struct{}

func (m MacDNS) Config(ip string) (err error) {
	var interfaces = macGetAllNetworkServices()

	for _, interface_name := range interfaces {
		if err = macDNSConfigForInterface(interface_name, ip); err != nil {
			return err
		}
	}
	return nil
}

func (m MacDNS) Deconfig() (err error) {
	var interfaces = macGetAllNetworkServices()

	for _, interface_name := range interfaces {
		if err = macDNSDeconfigForInterface(interface_name); err != nil {
			return err
		}
	}
	return nil
}

func (m MacDNS) FlushCache() (err error) {
	var cmd = exec.Command("dscacheutil", "-flushcache")
	var out strings.Builder
	var stderr strings.Builder

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	// TODO: do something about this sudo? Seems the only one that requires it...
	cmd = exec.Command("sudo", "killall", "-HUP", "mDNSResponder")
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return err
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
		return err
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
		fmt.Println(fmt.Sprint(err) + "3: " + stderr.String())
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
		return err
	}
	return nil
}
