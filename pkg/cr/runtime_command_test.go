package cr

import "testing"

func TestBoundedRuntimeOutputEnforcesLimit(t *testing.T) {
	output := &boundedRuntimeOutput{remaining: 4}
	written, err := output.Write([]byte("tests"))
	if err != nil {
		t.Fatalf("write bounded output: %v", err)
	}
	if written != 5 || output.String() != "test" || !output.truncated {
		t.Fatalf("unexpected bounded output: written=%d output=%q truncated=%t", written, output.String(), output.truncated)
	}
}
