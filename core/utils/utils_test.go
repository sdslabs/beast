package utils

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/sdslabs/beastv4/pkg/auth"
)

func TestGetTempIDsDeterministicAndBounded(t *testing.T) {
	shortImg := GetTempImageId("short")
	if len(shortImg) > 30 {
		t.Fatalf("short temp image id too long: %q (%d)", shortImg, len(shortImg))
	}

	longA := "very-long-challenge-name-with-shared-prefix-alpha"
	longB := "very-long-challenge-name-with-shared-prefix-beta"

	imgA := GetTempImageId(longA)
	imgB := GetTempImageId(longB)
	if len(imgA) > 30 || len(imgB) > 30 {
		t.Fatalf("temp image ids exceed 30 chars: %q (%d), %q (%d)", imgA, len(imgA), imgB, len(imgB))
	}
	if imgA == imgB {
		t.Fatalf("expected distinct temp image ids for distinct long names, both got %q", imgA)
	}
	if imgA != GetTempImageId(longA) {
		t.Fatalf("expected GetTempImageId to be deterministic")
	}

	cidA := GetTempContainerId(longA)
	cidB := GetTempContainerId(longB)
	if len(cidA) > 30 || len(cidB) > 30 {
		t.Fatalf("temp container ids exceed 30 chars: %q (%d), %q (%d)", cidA, len(cidA), cidB, len(cidB))
	}
	if cidA == cidB {
		t.Fatalf("expected distinct temp container ids for distinct long names, both got %q", cidA)
	}
}

func TestGetUserRawURLEncodedJWT(t *testing.T) {
	auth.Init(100, 32, 3600, "beast-test", "secret-test-jwt-key", []string{"author"}, []string{"admin"}, []string{"contestant"})

	// Test usernames whose payload lengths require non-multiple-of-4 unpadded Base64URL
	for _, wantUser := range []string{"a", "ab", "alice", "url-safe_user+1234"} {
		header := b64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payloadBytes, _ := json.Marshal(map[string]any{"usr": wantUser, "eml": "contestant"})
		payload := b64.RawURLEncoding.EncodeToString(payloadBytes)
		token := fmt.Sprintf("Bearer %s.%s.sig", header, payload)

		gotUser, err := GetUser(token)
		if err != nil {
			t.Fatalf("GetUser(%q) unexpected error: %v", wantUser, err)
		}
		if gotUser != wantUser {
			t.Fatalf("GetUser() = %q, want %q", gotUser, wantUser)
		}

		signed, err := auth.GenerateJWT(auth.CreateModel(wantUser, "pw", "contestant"))
		if err != nil {
			t.Fatalf("GenerateJWT(%q): %v", wantUser, err)
		}
		gotSignedUser, err := GetUser("Bearer " + signed)
		if err != nil {
			t.Fatalf("GetUser(signed %q) unexpected error: %v", wantUser, err)
		}
		if gotSignedUser != wantUser {
			t.Fatalf("GetUser(signed) = %q, want %q", gotSignedUser, wantUser)
		}
	}
}
