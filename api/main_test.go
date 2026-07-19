package api

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

func TestListenAddressValidatesPort(t *testing.T) {
	for _, port := range []string{"0", "65536", "not-a-port", "5005; shutdown"} {
		if _, err := listenAddress(port); err == nil {
			t.Fatalf("expected invalid port %q to fail", port)
		}
	}
	if address, err := listenAddress("5005"); err != nil || address != ":5005" {
		t.Fatalf("listenAddress returned %q, %v", address, err)
	}
}

func TestHTTPServerHasDefensiveTimeouts(t *testing.T) {
	server := newHTTPServer(":5005", http.NewServeMux())
	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatalf("missing HTTP timeouts: %+v", server)
	}
	if server.MaxHeaderBytes <= 0 || server.MaxHeaderBytes > 64<<10 {
		t.Fatalf("unexpected max header bytes: %d", server.MaxHeaderBytes)
	}
	if server.ReadHeaderTimeout > 30*time.Second {
		t.Fatalf("read header timeout is too permissive: %v", server.ReadHeaderTimeout)
	}
	if server.TLSConfig == nil || server.TLSConfig.MinVersion < tls.VersionTLS12 {
		t.Fatalf("missing minimum TLS version: %+v", server.TLSConfig)
	}
}
