package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sdslabs/beastv4/core"
	challengeConfig "github.com/sdslabs/beastv4/core/config"
)

func TestGenerateChallengeTemplateRefusesOverwrite(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, core.CHALLENGE_CONFIG_FILE_NAME)
	if err := os.WriteFile(configPath, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := generateChallengeTemplate(directory); err == nil {
		t.Fatal("expected existing template error")
	}
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "keep" {
		t.Fatalf("existing template was modified: %q", contents)
	}
}

func TestGenerateChallengeTemplateUsesPrivateConfig(t *testing.T) {
	directory := t.TempDir()
	if err := generateChallengeTemplate(directory); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(directory, core.CHALLENGE_CONFIG_FILE_NAME))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("template mode = %04o, want 0600", info.Mode().Perm())
	}
	configuration, err := challengeConfig.LoadChallengeConfig(filepath.Join(directory, core.CHALLENGE_CONFIG_FILE_NAME))
	if err != nil {
		t.Fatalf("generated configuration is not valid TOML: %v", err)
	}
	if configuration.Challenge.Metadata.Type != core.STATIC_CHALLENGE_TYPE_NAME || configuration.Challenge.Env.StaticContentDir != core.PUBLIC {
		t.Fatalf("unexpected generated challenge defaults: %#v", configuration.Challenge)
	}
}
