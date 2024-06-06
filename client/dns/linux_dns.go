package dns

import (
	"fmt"
	"os/exec"
	"strings"
)

// TODO: find some way to test this... won't work in a Docker container as it is
const TMPFILE string = "/tmp/resolv.conf"

type LinuxDNS struct{}

func (m LinuxDNS) Config(ip string) (err error) {
	// TODO: don't do this with bash lmao
	var cmd1 = exec.Command("/bin/sh", "-c", fmt.Sprintf("echo 'nameserver %s' | sudo tee /etc/resolv-manual.conf", ip))
	var cmd2 = exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo mv /etc/resolv.conf %s", TMPFILE))
	var cmd3 = exec.Command("/bin/sh", "-c", "sudo ln -s /etc/resolv-manual.conf /etc/resolv.conf")
	var out strings.Builder
	var stderr strings.Builder

	cmd1.Stdout = &out
	cmd1.Stderr = &stderr
	err = cmd1.Run()
	if err != nil {
		return err
	}
	cmd2.Stdout = &out
	cmd2.Stderr = &stderr
	err = cmd2.Run()
	if err != nil {
		return err
	}
	cmd3.Stdout = &out
	cmd3.Stderr = &stderr
	err = cmd3.Run()
	if err != nil {
		return err
	}

	return nil
}

func (m LinuxDNS) Deconfig() (err error) {
	// TODO: don't do this with bash lmao
	var cmd1 = exec.Command("/bin/sh", "-c", fmt.Sprintf("mv %s /etc/resolv.conf", TMPFILE))
	var cmd2 = exec.Command("/bin/sh", "-c", "rm /etc/resolv-manual.conf")
	var out strings.Builder
	var stderr strings.Builder

	cmd1.Stdout = &out
	cmd1.Stderr = &stderr
	err = cmd1.Run()
	if err != nil {
		return err
	}
	cmd2.Stdout = &out
	cmd2.Stderr = &stderr
	err = cmd2.Run()
	if err != nil {
		return err
	}

	return nil
}

func (m LinuxDNS) FlushCache() (err error) {
	return nil // TODO: implement cache flushing
}
