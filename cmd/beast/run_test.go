package main

import (
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
