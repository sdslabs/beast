package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileSelfAndTruncate(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(src, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Self-copy should be a safe no-op and preserve content.
	if err := CopyFile(src, src); err != nil {
		t.Fatalf("self CopyFile failed: %v", err)
	}
	got, err := os.ReadFile(src)
	if err != nil || string(got) != "hello world" {
		t.Fatalf("unexpected self-copy content %q, err=%v", string(got), err)
	}

	// Overwriting a longer destination file with a shorter source should truncate cleanly.
	dst := filepath.Join(tmp, "b.txt")
	if err := os.WriteFile(dst, []byte("very long previous content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile overwrite failed: %v", err)
	}
	gotDst, err := os.ReadFile(dst)
	if err != nil || string(gotDst) != "hello world" {
		t.Fatalf("unexpected overwritten content %q, err=%v", string(gotDst), err)
	}
}

func TestCopyDirectoryPreventsSelfRecursion(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CopyDirectory(src, src); err == nil {
		t.Fatal("expected error when copying directory into itself")
	}
	if err := CopyDirectory(src, filepath.Join(src, "static")); err == nil {
		t.Fatal("expected error when copying directory into its subdirectory")
	}

	dst := filepath.Join(tmp, "dst")
	if err := CopyDirectory(src, dst); err != nil {
		t.Fatalf("valid CopyDirectory failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "nested", "file.txt"))
	if err != nil || string(got) != "data" {
		t.Fatalf("unexpected copied content %q, err=%v", string(got), err)
	}
}
