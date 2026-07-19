package client

import "testing"

func TestAuthorizeRequiresHTTPS(t *testing.T) {
	if _, err := Authorize("password", "http://localhost:5005", "user", ""); err == nil {
		t.Fatal("expected plaintext authorization URL to fail")
	}
}
