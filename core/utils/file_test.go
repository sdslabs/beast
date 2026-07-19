package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
)

func TestGetChallengeDirFailsClosed(t *testing.T) {
	previousRoot := core.BEAST_GLOBAL_DIR
	previousConfig := config.Cfg
	core.BEAST_GLOBAL_DIR = t.TempDir()
	config.Cfg = &config.BeastConfig{}
	defer func() {
		core.BEAST_GLOBAL_DIR = previousRoot
		config.Cfg = previousConfig
	}()

	for _, name := range []string{"", "missing", "../escape", "/absolute"} {
		if path := GetChallengeDir(name); path != "" {
			t.Fatalf("GetChallengeDir(%q) = %q, want empty", name, path)
		}
	}

	want := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR, "valid-challenge")
	if err := os.MkdirAll(want, 0700); err != nil {
		t.Fatal(err)
	}
	if got := GetChallengeDir("valid-challenge"); got != want {
		t.Fatalf("GetChallengeDir() = %q, want %q", got, want)
	}
}
