package remoteManager

import (
	"errors"
	"testing"
)

func TestBoundedCommandOutputStopsAtLimit(t *testing.T) {
	output := &boundedCommandOutput{limit: 4}
	written, err := output.Write([]byte("abcdef"))
	if written != 4 || !errors.Is(err, errRemoteOutputLimit) {
		t.Fatalf("write = %d, %v", written, err)
	}
	if got := output.String(); got != "abcd" || !output.exceeded {
		t.Fatalf("output = %q, exceeded = %v", got, output.exceeded)
	}
}
