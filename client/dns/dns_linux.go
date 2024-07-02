package dns

import (
	"fmt"
	"os/exec"
	"strings"
)

// TODO: find some way to test this... won't work in a Docker container as it is
const TMPFILE string = "/tmp/resolv.conf"

func Config(ip string) (err error) {
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
		return fmt.Errorf("Failed to run [tee /etc/resolv.conf...]. Stderr: %s", stderr.String())
	}
	cmd2.Stdout = &out
	cmd2.Stderr = &stderr
	err = cmd2.Run()
	if err != nil {
		return fmt.Errorf("Failed to run [mv resolv.conf -> tmp...]. Stderr: %s", stderr.String())
	}
	cmd3.Stdout = &out
	cmd3.Stderr = &stderr
	err = cmd3.Run()
	if err != nil {
		return fmt.Errorf("Failed to run [ln -s resolv-manual resolv...]. Stderr: %s", stderr.String())
	}

	return nil
}

func Deconfig() (err error) {
	// TODO: don't do this with bash lmao
	var cmd1 = exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo mv %s /etc/resolv.conf", TMPFILE))
	var cmd2 = exec.Command("/bin/sh", "-c", "sudo rm /etc/resolv-manual.conf")
	var out strings.Builder
	var stderr strings.Builder

	cmd1.Stdout = &out
	cmd1.Stderr = &stderr
	err = cmd1.Run()
	if err != nil {
		return fmt.Errorf("Failed to run [mv tmp-resolv -> resolv...]. Stderr: %s", stderr.String())
	}
	cmd2.Stdout = &out
	cmd2.Stderr = &stderr
	err = cmd2.Run()
	if err != nil {
		return fmt.Errorf("Failed to run [rm resolv-manual...]. Stderr: %s", stderr.String())
	}

	return nil
}

func FlushCache() (err error) {
	// sudo resolvectl flush-caches

	// var cmd1 = exec.Command("/bin/sh", "-c", "sudo resolvectl flush-caches")
	// var out strings.Builder
	// var stderr strings.Builder

	// cmd1.Stdout = &out
	// cmd1.Stderr = &stderr
	// err = cmd1.Run()
	// if err != nil {
	// 	return fmt.Errorf("Failed to run [mv tmp-resolv -> resolv...]. Stderr: %s", stderr.String())
	// }

	return nil
}
