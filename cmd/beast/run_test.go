package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcquireControllerLockRejectsSecondController(t *testing.T) {
	directory := t.TempDir()
	first, err := acquireControllerLock(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseControllerLock(first)

	second, err := acquireControllerLock(directory)
	if second != nil {
		releaseControllerLock(second)
		t.Fatal("second controller acquired lock")
	}
	if err == nil || !strings.Contains(err.Error(), "already active") {
		t.Fatalf("expected active controller error, got %v", err)
	}
}

func TestLoadDefaultAuthorPasswordRequiresPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("a-secure-password\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDefaultAuthorPassword(path); err == nil {
		t.Fatal("expected public password file to fail")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	password, err := loadDefaultAuthorPassword(path)
	if err != nil {
		t.Fatal(err)
	}
	if password != "a-secure-password" {
		t.Fatalf("unexpected password %q", password)
	}
}

func TestControllerLockCanBeReacquiredAfterRelease(t *testing.T) {
	directory := t.TempDir()
	first, err := acquireControllerLock(directory)
	if err != nil {
		t.Fatal(err)
	}
	releaseControllerLock(first)

	second, err := acquireControllerLock(directory)
	if err != nil {
		t.Fatal(err)
	}
	releaseControllerLock(second)
}

func TestWriteJSONReplacesAtomicallyWithPrivatePermissions(t *testing.T) {
	directory := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	location := filepath.Join(directory, "cache.json")
	if err := os.Symlink(outside, location); err != nil {
		t.Fatal(err)
	}

	if err := writeJson(map[string]string{"state": "fresh"}, location); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(location)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != `{"state":"fresh"}` {
		t.Fatalf("unexpected JSON cache %q", contents)
	}
	outsideContents, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(outsideContents) != "keep" {
		t.Fatalf("symlink target was overwritten: %q", outsideContents)
	}
	info, err := os.Stat(location)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("cache permissions = %04o, want 0600", info.Mode().Perm())
	}
}
