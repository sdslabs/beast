package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sdslabs/beastv4/core"
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

func TestExampleGlobalConfigHasNoUnknownKeys(t *testing.T) {
	var config BeastConfig
	path := filepath.Join("..", "..", "_examples", "example.config.toml")
	if err := decodeTOMLFileStrict(path, &config); err != nil {
		t.Fatal(err)
	}
}

func TestExampleChallengeConfigsHaveNoUnknownKeys(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "_examples", "*", "beast.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no example challenge configs found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			if _, err := LoadChallengeConfig(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetAvailableChallengeTypesIsStable(t *testing.T) {
	first := GetAvailableChallengeTypes()
	second := GetAvailableChallengeTypes()
	if len(first) != len(second) {
		t.Fatalf("challenge types grew between calls: %d then %d", len(first), len(second))
	}
	seen := make(map[string]bool, len(first))
	for _, challengeType := range first {
		if seen[challengeType] {
			t.Fatalf("duplicate challenge type %q", challengeType)
		}
		seen[challengeType] = true
	}
}

func TestChallengeMetadataRejectsInvalidScoringAndLinks(t *testing.T) {
	tests := []ChallengeMetadata{
		{Name: "challenge", Flag: "flag", Type: "static", MinPoints: 200, MaxPoints: 100},
		{Name: "challenge", Flag: "flag", Type: "static", Points: 100, MinPoints: 200},
		{Name: "challenge", Flag: "flag", Type: "static", Points: 200, MaxPoints: 100},
		{Name: "challenge", Flag: "flag", Type: "static", PreReqs: []string{"../escape"}},
		{Name: "challenge", Flag: "flag", Type: "static", AdditionalLinks: []string{"javascript:alert(1)"}},
	}
	for _, metadata := range tests {
		if err, _ := metadata.ValidateRequiredFields(); err == nil {
			t.Fatalf("expected invalid metadata error: %+v", metadata)
		}
	}
}

func TestChallengeEnvRejectsInvalidPortsAndEnvironmentKeys(t *testing.T) {
	for _, ports := range [][]uint32{{0}, {80, 80}, {65536}} {
		env := ChallengeEnv{Ports: ports}
		if err := env.ExtractPorts(); err == nil {
			t.Fatalf("expected invalid ports error: %v", ports)
		}
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "value"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	previous := Cfg
	Cfg = &BeastConfig{AllowedBaseImages: []string{core.DEFAULT_BASE_IMAGE}}
	defer func() { Cfg = previous }()
	env := ChallengeEnv{
		Ports:           []uint32{8080},
		RunCmd:          "true",
		EnvironmentVars: []EnvironmentVar{{Key: "BAD-NAME", Value: "value"}},
	}
	if err := env.ValidateRequiredFields("bare", dir); err == nil || !strings.Contains(err.Error(), "environment variable key") {
		t.Fatalf("expected invalid environment key error, got %v", err)
	}
}

func TestAuthorRequiresCanonicalEmail(t *testing.T) {
	for _, email := range []string{"not-an-email", "Author <author@example.com>"} {
		author := Author{Email: email}
		if err := author.ValidateRequiredFields(); err == nil {
			t.Fatalf("expected invalid email error for %q", email)
		}
	}
}

func TestChallengeAssetsStayInsideStaticRoot(t *testing.T) {
	parent := t.TempDir()
	challenge := filepath.Join(parent, "challenge")
	static := filepath.Join(challenge, "static")
	if err := os.MkdirAll(static, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "secret"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	metadata := ChallengeMetadata{Assets: []string{"../../secret"}}
	if err := metadata.ValidateAssets(challenge, "static"); err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("expected escaping asset error, got %v", err)
	}
}

func TestGetDefaultPortHonorsConfiguredPort(t *testing.T) {
	env := ChallengeEnv{Ports: []uint32{8080, 9000}, DefaultPort: 9000}
	if got := env.GetDefaultPort(); got != 9000 {
		t.Fatalf("GetDefaultPort() = %d, want 9000", got)
	}
}

func TestUnknownServerDoesNotUseLocalDocker(t *testing.T) {
	config := BeastConfig{AvailableServers: map[string]AvailableServer{
		"localhost": {Host: "localhost", Active: true},
	}}
	if config.UseLocalDockerDaemon("missing") {
		t.Fatal("unknown server silently selected the local Docker daemon")
	}
}

func TestGitRemoteRejectsUnsafeIdentifiersAndURLs(t *testing.T) {
	tests := []GitRemote{
		{Url: "git@example.com:repo.git", RemoteName: "../../escape", Branch: "main", Secret: "key"},
		{Url: "git@example.com:repo.git", RemoteName: "origin", Branch: "../main", Secret: "key"},
		{Url: "https://example.com/repo.git", RemoteName: "origin", Branch: "main", Secret: "key"},
	}
	for _, remote := range tests {
		if err := remote.ValidateGitConfig(); err == nil {
			t.Fatalf("expected unsafe git remote error: %+v", remote)
		}
	}
}

func TestServerRejectsInvalidHostname(t *testing.T) {
	server := AvailableServer{Host: "host;id", PortRange: "10000:20000", Active: true}
	if err := server.ValidateServerConfig(); err == nil {
		t.Fatal("expected invalid hostname error")
	}
}

func TestDataStoreConfigRejectsUnsafeValues(t *testing.T) {
	psql := PsqlConfig{User: "beast", Password: "secret", Dbname: "../../escape", Host: "localhost", Port: "5432", SslMode: "prefer"}
	if err := psql.ValidatePsqlConfig(); err == nil {
		t.Fatal("expected unsafe database name error")
	}
	psql = PsqlConfig{User: "beast", Password: "secret", Dbname: "beast", Host: "localhost", Port: "5432", SslMode: "invalid"}
	if err := psql.ValidatePsqlConfig(); err == nil {
		t.Fatal("expected invalid sslmode error")
	}
	redis := RedisConfig{User: "beast", Host: "host;id", Port: "6379", Password: "secret"}
	if err := redis.ValidateRedisConfig(); err == nil {
		t.Fatal("expected invalid Redis host error")
	}
	redis = RedisConfig{User: "beast", Host: "localhost", Port: "0", Password: "secret"}
	if err := redis.ValidateRedisConfig(); err == nil {
		t.Fatal("expected invalid Redis port error")
	}
}
