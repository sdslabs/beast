package utils

import "testing"

func TestBoundedCommandOutputEnforcesLimit(t *testing.T) {
	output := &boundedCommandOutput{remaining: 4}
	written, err := output.Write([]byte("tests"))
	if err != nil {
		t.Fatal(err)
	}
	if written != 5 || output.String() != "test" || !output.truncated {
		t.Fatalf("unexpected bounded output: written=%d output=%q truncated=%t", written, output.String(), output.truncated)
	}
}
