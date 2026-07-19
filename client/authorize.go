package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const maxAuthorizeResponseBytes = 1 << 20

type Response struct {
	Message   string `json:"message"`
	Challenge []byte `json:"challenge"`
	Token     string `json:"token"`
}

func Authorize(password, host, username, caFile string) (Response, error) {
	base, err := url.Parse(host)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil {
		return Response{}, fmt.Errorf("Beast host must be a valid HTTPS URL")
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile != "" {
		certificate, err := os.ReadFile(caFile)
		if err != nil {
			return Response{}, fmt.Errorf("read CA file: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(certificate) {
			return Response{}, fmt.Errorf("CA file contains no certificates")
		}
		tlsConfig.RootCAs = roots
	}
	httpClient := &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	endpoint := base.ResolveReference(&url.URL{Path: "auth/login"})
	response, err := httpClient.PostForm(endpoint.String(), url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		return Response{}, fmt.Errorf("authorize request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxAuthorizeResponseBytes+1))
	if err != nil {
		return Response{}, fmt.Errorf("read authorize response: %w", err)
	}
	if len(body) > maxAuthorizeResponseBytes {
		return Response{}, fmt.Errorf("authorize response exceeds 1 MiB")
	}
	if response.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("authorization failed with HTTP status %d", response.StatusCode)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return Response{}, fmt.Errorf("parse authorize response: %w", err)
	}
	if result.Token == "" {
		return Response{}, fmt.Errorf("authorize response did not contain a token")
	}
	return result, nil
}
