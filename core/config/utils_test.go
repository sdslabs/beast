package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadChallengeConfigRejectsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "beast.toml")
	data := []byte("[author]\nemail = \"author@example.com\"\nunknown = true\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadChallengeConfig(path)
	if err == nil || !strings.Contains(err.Error(), "author.unknown") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestLoadChallengeConfigAcceptsKnownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "beast.toml")
	data := []byte("[author]\nemail = \"author@example.com\"\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	config, err := LoadChallengeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Author.Email != "author@example.com" {
		t.Fatalf("unexpected author email %q", config.Author.Email)
	}
}

func TestChallengeEnvRejectsEscapingPaths(t *testing.T) {
	parent := t.TempDir()
	challengeDir := filepath.Join(parent, "challenge")
	if err := os.Mkdir(challengeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	env := ChallengeEnv{StaticContentDir: "../outside"}
	err := env.ValidateRequiredFields("static", challengeDir)
	if err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("expected escaping path error, got %v", err)
	}
}

func TestChallengeEnvRejectsEscapingComposeSymlink(t *testing.T) {
	parent := t.TempDir()
	challengeDir := filepath.Join(parent, "challenge")
	if err := os.Mkdir(challengeDir, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "compose.yml")
	if err := os.WriteFile(outside, []byte("services: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(challengeDir, "compose.yml")); err != nil {
		t.Fatal(err)
	}

	env := ChallengeEnv{DockerCompose: "compose.yml"}
	err := env.ValidateRequiredFields("bare", challengeDir)
	if err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("expected escaping symlink error, got %v", err)
	}
}

func TestChallengeMetadataRejectsUnsafeNames(t *testing.T) {
	tests := []string{"../../escape", "name; touch pwned", "UPPERCASE", strings.Repeat("a", 65)}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			metadata := ChallengeMetadata{Name: name, Flag: "flag", Type: "static"}
			err, _ := metadata.ValidateRequiredFields()
			if err == nil || !strings.Contains(err.Error(), "must match") {
				t.Fatalf("expected unsafe name error, got %v", err)
			}
		})
	}
}
