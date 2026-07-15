package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTarRejectsSymlinks(t *testing.T) {
	parent := t.TempDir()
	contextDir := filepath.Join(parent, "challenge")
	destinationDir := filepath.Join(parent, "staging")
	if err := os.Mkdir(contextDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destinationDir, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "secret")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(contextDir, "link")); err != nil {
		t.Fatal(err)
	}

	err := Tar(contextDir, Gzip, destinationDir, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported file type") {
		t.Fatalf("expected unsupported file error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(destinationDir, "challenge.tar.gz")); !os.IsNotExist(statErr) {
		t.Fatalf("partial archive was not removed: %v", statErr)
	}
}
