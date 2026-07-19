package probes

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHTTPProberVerifiesTLSByDefault(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := NewHTTPProber().Probe(endpoint, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result != Failure {
		t.Fatalf("untrusted TLS endpoint result = %s, want failure", result)
	}
}

func TestHTTPProberBoundsResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(strings.Repeat("x", maxProbeBody+1)))
	}))
	defer server.Close()
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = NewWithTLSConfig(&tls.Config{}).Probe(endpoint, nil, time.Second)
	if err == nil {
		t.Fatal("expected oversized response error")
	}
}
