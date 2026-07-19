package manager

import (
	"testing"

	"github.com/sdslabs/beastv4/core/config"
)

func TestSyncWithoutActiveRemotesSucceeds(t *testing.T) {
	previous := config.Cfg
	config.Cfg = &config.BeastConfig{}
	t.Cleanup(func() { config.Cfg = previous })

	if err := SyncBeastRemote(""); err != nil {
		t.Fatalf("SyncBeastRemote() = %v, want nil", err)
	}
	if err := ResetBeastRemote(""); err != nil {
		t.Fatalf("ResetBeastRemote() = %v, want nil", err)
	}
}
