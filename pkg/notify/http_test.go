package notify

import (
	"net"
	"testing"
)

func TestIsPublicIPRejectsInternalDestinations(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fe80::1"} {
		if isPublicIP(net.ParseIP(address)) {
			t.Fatalf("expected %s to be rejected", address)
		}
	}
	if !isPublicIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("expected public address to be accepted")
	}
}
