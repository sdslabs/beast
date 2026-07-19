package config

import (
	"testing"

	"github.com/sdslabs/beastv4/core"
)

func TestInitConfigReturnsLoadError(t *testing.T) {
	previousCfg := Cfg
	previousGlobalDir := core.BEAST_GLOBAL_DIR
	Cfg = nil
	core.BEAST_GLOBAL_DIR = t.TempDir()
	t.Cleanup(func() {
		Cfg = previousCfg
		core.BEAST_GLOBAL_DIR = previousGlobalDir
	})

	if err := InitConfig(); err == nil {
		t.Fatal("expected missing configuration error")
	}
	if Cfg != nil {
		t.Fatal("configuration was initialized after a load error")
	}
}
