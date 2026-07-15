package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadChallengeConfigRejectsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "beast.toml")
	data := []byte("[author]\nemail = \"author@example.com\"\nunknown = true\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadChallengeConfig(path)
	if err == nil || !strings.Contains(err.Error(), "author.unknown") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestLoadChallengeConfigAcceptsKnownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "beast.toml")
	data := []byte("[author]\nemail = \"author@example.com\"\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	config, err := LoadChallengeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Author.Email != "author@example.com" {
		t.Fatalf("unexpected author email %q", config.Author.Email)
	}
}
