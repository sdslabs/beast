package cr

import (
	"errors"
	"testing"
)

func TestBoundedBuildOutputStopsAtLimit(t *testing.T) {
	output := &boundedBuildOutput{}
	data := make([]byte, maxBuildOutput+1)
	written, err := output.Write(data)
	if written != maxBuildOutput || !errors.Is(err, errBuildOutputLimit) {
		t.Fatalf("write = %d, %v", written, err)
	}
	if !output.exceeded || output.Buffer().Len() != maxBuildOutput {
		t.Fatalf("output limit was not recorded")
	}
}
