package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaintainedExamplesValidate(t *testing.T) {
	previous := Cfg
	Cfg = &BeastConfig{
		AllowedBaseImages: []string{"ubuntu:24.04", "debian:bookworm"},
		CPUShares:         1024,
		CPUsLimit:         1,
		Memory:            1 << 30,
		PidsLimit:         256,
	}
	defer func() { Cfg = previous }()

	examplesRoot := filepath.Join("..", "..", "_examples")
	entries, err := os.ReadDir(examplesRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		challengeDir := filepath.Join(examplesRoot, entry.Name())
		configPath := filepath.Join(challengeDir, "beast.toml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			configuration, err := LoadChallengeConfig(configPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := configuration.ValidateRequiredFields(challengeDir); err != nil {
				t.Fatal(err)
			}
		})
	}
}
