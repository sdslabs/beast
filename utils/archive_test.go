package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTarUsesRelativeEntryNames(t *testing.T) {
	parent := t.TempDir()
	contextDir := filepath.Join(parent, "challenge")
	destinationDir := filepath.Join(parent, "staging")
	if err := os.Mkdir(contextDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destinationDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextDir, "file.txt"), []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Tar(contextDir, Gzip, destinationDir, nil, nil); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filepath.Join(destinationDir, "challenge.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if filepath.IsAbs(header.Name) || strings.HasPrefix(header.Name, "../") {
			t.Fatalf("archive contains unsafe entry name %q", header.Name)
		}
	}
}

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

func TestTarFailsWhenAdditionalContextIsMissing(t *testing.T) {
	parent := t.TempDir()
	contextDir := filepath.Join(parent, "challenge")
	destinationDir := filepath.Join(parent, "staging")
	if err := os.Mkdir(contextDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destinationDir, 0700); err != nil {
		t.Fatal(err)
	}

	err := Tar(contextDir, Gzip, destinationDir, map[string]string{
		"Dockerfile": filepath.Join(parent, "missing"),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid additional archive file") {
		t.Fatalf("expected missing additional file error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(destinationDir, "challenge.tar.gz")); !os.IsNotExist(statErr) {
		t.Fatalf("partial archive was not removed: %v", statErr)
	}
}

func TestExtractTarGzipRejectsTraversal(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "unsafe.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	contents := []byte("secret")
	if err := tarWriter.WriteHeader(&tar.Header{Name: "../escape", Mode: 0600, Size: int64(len(contents))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(t.TempDir(), "extracted")
	if err := ExtractTarGzip(archivePath, destination); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("partial extraction was not removed: %v", err)
	}
}
