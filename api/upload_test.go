package api

import (
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/sdslabs/beastv4/core"
)

func TestChallengeArchiveFilenameRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{"", ".zip", "challenge.tar", "../challenge.zip", `dir\\challenge.zip`} {
		if _, err := challengeArchiveFilename(name); err == nil {
			t.Fatalf("expected filename %q to be rejected", name)
		}
	}
	if got, err := challengeArchiveFilename("challenge.ZIP"); err != nil || got != "challenge.ZIP" {
		t.Fatalf("expected valid ZIP name, got %q, %v", got, err)
	}
}

func TestSaveChallengeArchiveRejectsOversizeHeader(t *testing.T) {
	header := &multipart.FileHeader{Filename: "challenge.zip", Size: maxChallengeUploadBytes + 1}
	err := saveChallengeArchive(header, filepath.Join(t.TempDir(), "challenge.zip"))
	if !errors.Is(err, errChallengeUploadTooLarge) {
		t.Fatalf("expected upload size error, got %v", err)
	}
}

func TestPersistUploadedChallengeDoesNotReplaceExistingChallenge(t *testing.T) {
	previousGlobalDir := core.BEAST_GLOBAL_DIR
	core.BEAST_GLOBAL_DIR = t.TempDir()
	t.Cleanup(func() { core.BEAST_GLOBAL_DIR = previousGlobalDir })

	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "beast.toml"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := persistUploadedChallenge(source, "challenge"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "beast.toml"), []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := persistUploadedChallenge(source, "challenge"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected existing challenge error, got %v", err)
	}

	stored := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR, "challenge", "beast.toml")
	contents, err := os.ReadFile(stored)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "original" {
		t.Fatalf("existing challenge was replaced: %q", contents)
	}
}
