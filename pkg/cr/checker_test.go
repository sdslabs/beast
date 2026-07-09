package cr

import (
	"strings"
	"testing"
)

func TestSadServersCheckerContainerNameSanitizesInstanceID(t *testing.T) {
	name := SadServersCheckerContainerName("Instance/One!")
	if !strings.HasPrefix(name, "beast-checker-instance-one--") {
		t.Fatalf("unexpected checker container name %q", name)
	}
	if strings.ContainsAny(name, "/!") {
		t.Fatalf("checker container name contains unsafe characters: %q", name)
	}

	other := SadServersCheckerContainerName("Instance/One!")
	if other == name {
		t.Fatalf("expected checker container names to include unique suffixes")
	}
}
