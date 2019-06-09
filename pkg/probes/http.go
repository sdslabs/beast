package probes

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var defaultTransport = http.DefaultTransport.(*http.Transport)

// New creates Prober that will skip TLS verification while probing.
func NewHTTPProber() HttpProber {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	return NewWithTLSConfig(tlsConfig)
}

func setOldTransportDefaults(t *http.Transport) *http.Transport {
	if t.DialContext == nil && t.Dial == nil {
		t.DialContext = defaultTransport.DialContext
	}

	if t.TLSHandshakeTimeout == 0 {
		t.TLSHandshakeTimeout = defaultTransport.TLSHandshakeTimeout
	}
	return t
}

func NewWithTLSConfig(config *tls.Config) HttpProber {
	transport := setOldTransportDefaults(
		&http.Transport{
			TLSClientConfig:   config,
			DisableKeepAlives: true,
		})
	return HttpProber{transport}
}

type HttpProber struct {
	transport *http.Transport
}

// Probe returns a ProbeRunner capable of running an HTTP check.
// If the HTTP response code is successful (i.e. 400 > code >= 200), it returns Success.
// If the HTTP response code is unsuccessful or HTTP communication fails, it returns Failure.
func (pr HttpProber) Probe(url *url.URL, headers http.Header, timeout time.Duration) (ProbeResult, string, error) {
	client := &http.Client{
		Timeout:   timeout,
		Transport: pr.transport,
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return Failure, err.Error(), nil
	}

	req.Header = headers
	if headers.Get("Host") != "" {
		req.Host = headers.Get("Host")
	}
	res, err := client.Do(req)
	if err != nil {
		return Failure, err.Error(), nil
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Failure, "", err
	}
	body := string(b)

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusBadRequest {
		if res.StatusCode >= http.StatusMultipleChoices { // Redirect
			return Warning, body, nil
		}
		return Success, body, nil
	}

	return Failure, fmt.Sprintf("HTTP probe failed with statuscode: %d", res.StatusCode), nil
}
