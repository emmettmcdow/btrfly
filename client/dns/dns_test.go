package dns

import (
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

var d DNSConfig = ConfigAgent()

type callback_t func() error

func DNSLookupHelper(t *testing.T, callback callback_t) {
	// Should pass
	err := callback()
	if err != nil {
		t.Error("Failed to lookup DNS. System is not properly configured")
	}

	d.Config(BLACKHOLE_IP)
	// Should fail
	err = callback()
	if err == nil {
		d.Deconfig()
		t.Error("Successfully looked up the IP, meaning MacDNSConfig failed.")
	}

	d.Deconfig()
	d.FlushCache()
	// Should pass
	err = callback()
	if err != nil {
		t.Error("Failed to look up the IP. You should see about fixing the " +
			"configuration manually because MacDNSDeconfig is wildin'.")
	}
}

func TestGolangDNS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Platform is not darwin")
	}
	DNSLookupHelper(t, func() (err error) {
		_, err = net.LookupIP("google.com")
		return err
	})
}

func TestDNSNslookup(t *testing.T) {
	if runtime.GOOS != "darwin" {
		d.FlushCache()
		t.Skip("Platform is not darwin")
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
	if runtime.GOOS != "darwin" {
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
	if runtime.GOOS != "darwin" {
		t.Skip("Platform is not darwin")
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
