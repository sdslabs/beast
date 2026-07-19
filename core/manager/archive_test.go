package manager

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

type archiveEntry struct {
	name string
	mode os.FileMode
	body string
}

func writeTestArchive(t *testing.T, entries []archiveEntry) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "challenge.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(entry.mode)
		part, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func TestUnzipChallengeFolderExtractsRegularFiles(t *testing.T) {
	archivePath := writeTestArchive(t, []archiveEntry{{
		name: "challenge/beast.toml",
		mode: 0600,
		body: "[challenge]",
	}})
	destination := t.TempDir()

	extracted, err := UnzipChallengeFolder(archivePath, destination)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(extracted, "challenge", "beast.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "[challenge]" {
		t.Fatalf("unexpected contents: %q", contents)
	}
}

func TestUnzipChallengeFolderRejectsUnsafeEntries(t *testing.T) {
	tests := []archiveEntry{
		{name: "../escape", mode: 0600, body: "bad"},
		{name: "/absolute", mode: 0600, body: "bad"},
		{name: `windows\\escape`, mode: 0600, body: "bad"},
		{name: "link", mode: os.ModeSymlink | 0777, body: "target"},
	}
	for _, entry := range tests {
		t.Run(entry.name, func(t *testing.T) {
			archivePath := writeTestArchive(t, []archiveEntry{entry})
			if _, err := UnzipChallengeFolder(archivePath, t.TempDir()); err == nil {
				t.Fatal("expected unsafe archive entry to be rejected")
			}
		})
	}
}

func TestUnzipChallengeFolderRejectsDuplicatePaths(t *testing.T) {
	archivePath := writeTestArchive(t, []archiveEntry{
		{name: "challenge/flag", mode: 0600, body: "first"},
		{name: "challenge/flag", mode: 0600, body: "second"},
	})
	if _, err := UnzipChallengeFolder(archivePath, t.TempDir()); err == nil {
		t.Fatal("expected duplicate archive path to be rejected")
	}
}

func TestCopyDirRejectsSymlinksAndExistingDestinations(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("file", filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(source, filepath.Join(t.TempDir(), "copy")); err == nil {
		t.Fatal("expected symbolic link to be rejected")
	}
	if err := CopyDir(source, t.TempDir()); err == nil {
		t.Fatal("expected existing destination to be rejected")
	}
}
