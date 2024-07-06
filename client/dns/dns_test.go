package dns

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"testing"
)

// We only want to test for "Was the DNS configured properly?"
// We do not care if the DNS actually worked.
// We cannot be bothered to actually set up a DNS server here.
// Therefore, we will treat failures as successes and successes
// as failures. If we configure a nonsense DNS, and we fail to connect
// then we know we succeeded in configuring.

// IPv4 Black Hole
// https://superuser.com/questions/698244/ip-address-that-is-the-equivalent-of-dev-null
const BLACKHOLE_IP = "240.0.0.0"

type callback_t func() error

func DNSLookupHelper(t *testing.T, callback callback_t) {
	// Should pass
	err := callback()
	if err != nil {
		t.Error("Failed to lookup. System is not properly configured")
		return
	}

	err = Config(BLACKHOLE_IP)
	if err != nil {
		Deconfig()
		FlushCache()
		t.Error(fmt.Sprintf("Failed to Configure: %s", err))
		return
	}
	err = FlushCache()
	if err != nil {
		t.Error(fmt.Sprintf("Failed to FlushCache: %s", err))
		return
	}

	// Should fail
	err = callback()
	if err == nil {
		Deconfig()
		FlushCache()
		t.Error("Successfully looked up the IP, meaning dns.Config failed.")
		return
	}

	err = Deconfig()
	if err != nil {
		FlushCache()
		t.Error(fmt.Sprintf("Failed to Deconfigure: %s", err))
		return
	}
	err = FlushCache()
	if err != nil {
		t.Error(fmt.Sprintf("Failed to FlushCache: %s", err))
		return
	}
	// Should pass
	err = callback()
	if err != nil {
		t.Error("Failed to look up the IP. You should see about fixing the " +
			"configuration manually because dns.Deconfig is wildin'.")
		return
	}
}

func TestGolangDNS(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Platform is not supported")
	}
	DNSLookupHelper(t, func() (err error) {
		// This is an unfortunate side effect of the fact that Go checks the /etc/resolv.conf file
		// once every 5 seconds.
		// https://github.com/golang/go/blob/ad77cefeb2f5b3f1cef4383e974195ffc8610236/src/net/dnsclient_unix.go#L401
		// Need to make a new resolver every time in order to refresh
		resolver := net.Resolver{}
		_, err = resolver.LookupHost(context.TODO(), "google.com")
		return err
	})
}

func TestDNSCurl(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Platform is not supported")
	}
	_, err := exec.LookPath("curl")
	if err != nil {
		t.Skip("curl not installed on system")
	}

	DNSLookupHelper(t, func() (err error) {
		cmd := exec.Command("curl", "google.com")
		err = cmd.Run()
		return err
	})
}

func TestDNSNslookup(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Platform is not supported")
	}
	_, err := exec.LookPath("nslookup")
	if err != nil {
		t.Skip("nslookup not installed on system")
	}

	DNSLookupHelper(t, func() (err error) {
		cmd := exec.Command("nslookup", "google.com")
		err = cmd.Run()
		return err
	})
}

func TestDNSDig(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Platform is not darwin")
	}
	_, err := exec.LookPath("dig")
	if err != nil {
		t.Skip("dig not installed on system")
	}

	DNSLookupHelper(t, func() (err error) {
		cmd := exec.Command("dig", "google.com")
		err = cmd.Run()
		return err
	})
}
