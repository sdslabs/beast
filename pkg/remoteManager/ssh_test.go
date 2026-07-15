package remoteManager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sdslabs/beastv4/core/config"
)

func TestCreateSSHClientRejectsMissingKnownHosts(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(keyFile, []byte("not-a-key"), 0600); err != nil {
		t.Fatal(err)
	}
	server := config.AvailableServer{
		Active:         true,
		SSHKeyPath:     keyFile,
		KnownHostsFile: filepath.Join(t.TempDir(), "missing"),
	}

	_, err := CreateSSHClient(server)
	if err == nil || !strings.Contains(err.Error(), "known_hosts") {
		t.Fatalf("expected SSH configuration error, got %v", err)
	}
}
