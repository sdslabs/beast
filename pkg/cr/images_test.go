package cr

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSadServersCheckerContextArchiveFiltersChallengeFiles(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "challenge.tar.gz")
	writeTestTarGz(t, sourcePath, map[string]string{
		"check.sh":          "#!/bin/sh\nexit 0\n",
		"checker/helper.sh": "#!/bin/sh\nexit 0\n",
		"secret.txt":        "do not include\n",
		"nested/check.sh":   "#!/bin/sh\nexit 1\n",
	})

	destinationPath := filepath.Join(dir, "checker-context.tar.gz")
	if err := WriteSadServersCheckerContextArchive(sourcePath, destinationPath); err != nil {
		t.Fatalf("write checker context archive: %v", err)
	}

	entries := readTestTarGz(t, destinationPath)
	for _, want := range []string{"Dockerfile", "checker", "check.sh", "checker/helper.sh"} {
		if _, ok := entries[want]; !ok {
			t.Fatalf("expected checker context to include %s, got %#v", want, entries)
		}
	}
	for _, forbidden := range []string{"secret.txt", "nested/check.sh"} {
		if _, ok := entries[forbidden]; ok {
			t.Fatalf("checker context unexpectedly included %s", forbidden)
		}
	}
	if entries["check.sh"].mode != 0555 {
		t.Fatalf("expected check.sh mode 0555, got %#o", entries["check.sh"].mode)
	}
}

type testTarEntry struct {
	mode int64
	body string
}

func writeTestTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar gzip: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, body := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(body)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header %s: %v", name, err)
		}
		if _, err := tarWriter.Write([]byte(body)); err != nil {
			t.Fatalf("write tar body %s: %v", name, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close tar gzip: %v", err)
	}
}

func readTestTarGz(t *testing.T, path string) map[string]testTarEntry {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open tar gzip: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("open gzip reader: %v", err)
	}
	defer gzipReader.Close()

	entries := map[string]testTarEntry{}
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar entry: %v", err)
		}

		body := ""
		if header.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tarReader)
			if err != nil {
				t.Fatalf("read tar body %s: %v", header.Name, err)
			}
			body = string(data)
		}
		entries[header.Name] = testTarEntry{mode: header.Mode, body: body}
	}

	return entries
}
