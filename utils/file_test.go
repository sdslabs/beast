package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathWithinRejectsTraversal(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "challenge")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "secret")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := ResolvePathWithin(root, "../secret"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestResolvePathWithinRejectsEscapingSymlink(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "challenge")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "secret")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	if _, err := ResolvePathWithin(root, "link"); err == nil {
		t.Fatal("expected escaping symlink to be rejected")
	}
}

func TestResolvePathWithinAcceptsContainedPath(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "nested", "file")
	if err := os.Mkdir(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolvePathWithin(root, "nested/file")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != file {
		t.Fatalf("resolved %q, want %q", resolved, file)
	}
}

func TestValidateSecretFileRequiresPrivateRegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSecretFile(path); err == nil {
		t.Fatal("expected insecure permissions error")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSecretFile(path); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(dir, "link")
	if err := os.Symlink(path, symlink); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSecretFile(symlink); err == nil {
		t.Fatal("expected symlink error")
	}
}

func TestExpandHomePathOnlyExpandsHomePrefix(t *testing.T) {
	t.Setenv("HOME", "/tmp/beast-home")
	for _, input := range []string{"~/secret", "$HOME/secret", "${HOME}/secret"} {
		got, err := ExpandHomePath(input)
		if err != nil {
			t.Fatal(err)
		}
		if got != "/tmp/beast-home/secret" {
			t.Fatalf("ExpandHomePath(%q) = %q", input, got)
		}
	}
	got, err := ExpandHomePath("$UNTRUSTED/secret")
	if err != nil || got != "$UNTRUSTED/secret" {
		t.Fatalf("unexpected arbitrary expansion: %q, %v", got, err)
	}
}

func TestCopyDirectoryRejectsSymlinks(t *testing.T) {
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "copied")
	secret := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secret, []byte("host secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(source, "asset")); err != nil {
		t.Fatal(err)
	}
	if err := CopyDirectory(source, target); err == nil {
		t.Fatal("expected symlink rejection")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("partial destination was not removed: %v", err)
	}
}

func TestCopyFileRefusesExistingDestination(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source")
	destination := filepath.Join(directory, "destination")
	if err := os.WriteFile(source, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(source, destination); err == nil {
		t.Fatal("expected existing destination error")
	}
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "keep" {
		t.Fatalf("destination was overwritten: %q", contents)
	}
}
