package utils

import "testing"

func TestDockerProjectNameHelpers(t *testing.T) {
	challengeName := "Web Challenge 01"
	encoded := EncodeID(challengeName)

	if got, want := ProjectNameNotInstanced(challengeName), "beast-"+encoded; got != want {
		t.Fatalf("ProjectNameNotInstanced() = %q, want %q", got, want)
	}

	if got, want := ComposeDockerProjectNameInstanced(challengeName, "abc123"), "beast-instance-"+encoded+"-abc123"; got != want {
		t.Fatalf("ComposeDockerProjectNameInstanced() = %q, want %q", got, want)
	}
}
