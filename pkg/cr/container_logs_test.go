package cr

import (
	"errors"
	"strings"
	"testing"
)

func TestReadContainerLogsEnforcesLimit(t *testing.T) {
	logs, err := readContainerLogs(strings.NewReader("test"), 4)
	if err != nil {
		t.Fatalf("read logs at limit: %v", err)
	}
	if string(logs) != "test" {
		t.Fatalf("unexpected logs: %q", logs)
	}

	_, err = readContainerLogs(strings.NewReader("tests"), 4)
	if !errors.Is(err, errContainerLogLimit) {
		t.Fatalf("expected log limit error, got %v", err)
	}
}
