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

func TestLoadBeastConfigRejectsInsecurePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadBeastConfig(path)
	if err == nil || !strings.Contains(err.Error(), "must be 0600") {
		t.Fatalf("expected permissions error, got %v", err)
	}
}

func TestLoadBeastConfigRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.toml")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}

	_, err := LoadBeastConfig(path)
	if err == nil || !strings.Contains(err.Error(), "must not be a symbolic link") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestResourcesCannotExceedGlobalLimits(t *testing.T) {
	previous := Cfg
	Cfg = &BeastConfig{CPUShares: 100, Memory: 1024, PidsLimit: 10, CPUsLimit: 1}
	defer func() { Cfg = previous }()

	tests := []Resources{
		{CPUShares: 101},
		{Memory: 1025},
		{PidsLimit: 11},
		{CPUsLimit: 1.1},
	}
	for _, resources := range tests {
		if err := resources.ValidateRequiredFields(); err == nil {
			t.Fatalf("expected global limit error for %+v", resources)
		}
	}
}

func TestResourcesUseGlobalDefaults(t *testing.T) {
	previous := Cfg
	Cfg = &BeastConfig{CPUShares: 100, Memory: 1024, PidsLimit: 10, CPUsLimit: 1}
	defer func() { Cfg = previous }()

	resources := Resources{}
	if err := resources.ValidateRequiredFields(); err != nil {
		t.Fatal(err)
	}
	if resources.CPUShares != 100 || resources.Memory != 1024 || resources.PidsLimit != 10 || resources.CPUsLimit != 1 {
		t.Fatalf("unexpected defaults: %+v", resources)
	}
}

func TestServerConfigRequiresTLSFiles(t *testing.T) {
	if err := (&ServerConfig{}).Validate(); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required TLS files error, got %v", err)
	}
}

func TestServerConfigExpandsHomePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	server := ServerConfig{TLSCertFile: "$HOME/cert.pem", TLSKeyFile: "${HOME}/key.pem"}
	if err := server.Validate(); err == nil {
		t.Fatal("expected missing certificate error")
	}
	if server.TLSCertFile != filepath.Join(home, "cert.pem") || server.TLSKeyFile != filepath.Join(home, "key.pem") {
		t.Fatalf("paths were not expanded: %+v", server)
	}
}
