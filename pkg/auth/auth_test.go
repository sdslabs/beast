package auth

import (
	"bytes"
	"testing"
)

func TestCreateModelUsesUniqueSalt(t *testing.T) {
	Init(100, 32, 60, "issuer", "secret", nil, nil, nil)
	first, err := CreateModel("alice", "password", "user")
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreateModel("alice", "password", "user")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first.Salt, second.Salt) || bytes.Equal(first.Password, second.Password) {
		t.Fatal("password models reused salt-derived data")
	}
}
