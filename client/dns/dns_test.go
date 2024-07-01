package dns

import (
	"net"
	"os/exec"
	"runtime"
	"testing"
	"fmt"
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
	}

	err = Config(BLACKHOLE_IP)
	if err != nil {
		Deconfig()
		FlushCache()
		t.Error(fmt.Sprintf("Failed to Configure: %s", err))
	}
	// Should fail
	err = callback()
	if err == nil {
		Deconfig()
		FlushCache()
		t.Error("Successfully looked up the IP, meaning dns.Config failed.")
	}

	err = Deconfig()
	if err != nil {
		FlushCache()
		t.Error(fmt.Sprintf("Failed to Deconfigure: %s", err))
	}
	err = FlushCache()
	if err != nil {
		t.Error(fmt.Sprintf("Failed to FlushCache: %s", err))
	}
	// Should pass
	err = callback()
	if err != nil {
		t.Error("Failed to look up the IP. You should see about fixing the " +
			"configuration manually because dns.Deconfig is wildin'.")
	}
}

func TestGolangDNS(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Platform is not supported")
	}
	DNSLookupHelper(t, func() (err error) {
		_, err = net.LookupIP("google.com")
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
