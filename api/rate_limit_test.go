package api

import (
	"strings"
	"testing"
)

func TestRateLimitKeyDoesNotExposeIdentity(t *testing.T) {
	key := rateLimitKey("login-user", "sensitive-user")
	if strings.Contains(key, "sensitive-user") {
		t.Fatalf("rate limit key leaks identity: %q", key)
	}
	if key != rateLimitKey("login-user", "sensitive-user") {
		t.Fatal("rate limit key is not deterministic")
	}
}
