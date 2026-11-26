package config

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sdslabs/beastv4/core"
	log "github.com/sirupsen/logrus"
)

func TestValidateGitConfig(t *testing.T) {
	httpsRemote := GitRemote{
		Url:        "https://github.com/sdslabs/beast-challenges.git",
		RemoteName: "origin",
		Branch:     "main",
	}
	if err := httpsRemote.ValidateGitConfig(); err != nil {
		t.Fatalf("expected HTTPS remote without ssh_key to validate, got: %v", err)
	}

	sshMissingKey := GitRemote{
		Url:        "git@github.com:sdslabs/beast-challenges.git",
		RemoteName: "origin",
		Branch:     "main",
	}
	if err := sshMissingKey.ValidateGitConfig(); err == nil {
		t.Fatalf("expected SSH remote without ssh_key to fail validation")
	}

	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyPath, []byte("dummy"), 0600); err != nil {
		t.Fatalf("write temp key: %v", err)
	}
	sshWithKey := GitRemote{
		Url:        "git@github.com:sdslabs/beast-challenges.git",
		RemoteName: "origin",
		Branch:     "main",
		Secret:     keyPath,
	}
	if err := sshWithKey.ValidateGitConfig(); err != nil {
		t.Fatalf("expected SSH remote with existing ssh_key to validate, got: %v", err)
	}

	invalidRemote := GitRemote{
		Url:        "not a valid url",
		RemoteName: "origin",
	}
	if err := invalidRemote.ValidateGitConfig(); err == nil {
		t.Fatalf("expected invalid git url to fail validation")
	}
}

func TestMailConfigApplyEnvOverrides(t *testing.T) {
	t.Setenv("SMTP_FROM", "ctf@example.test")
	t.Setenv("SMTP_PASSWORD", "secret")
	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("SMTP_PORT", "51025")

	var mail MailConfig
	mail.ApplyEnvOverrides()

	if mail.From != "ctf@example.test" || mail.Password != "secret" || mail.SMTPHost != "localhost" || mail.SMTPPort != "51025" {
		t.Fatalf("unexpected MailConfig after ApplyEnvOverrides: %+v", mail)
	}
}

func TestValidateInstancedComposeSSHContractRequiresPortVariables(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("version: '3.8'\nservices:\n  app:\n    image: ubuntu:24.04\n"), 0644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	env := ChallengeEnv{
		DockerCompose: "docker-compose.yml",
	}
	if err := env.ValidateInstancedComposeSSHContract(dir); err == nil {
		t.Fatalf("expected error when PortVariables is empty")
	}
}

func TestChallengeEnvPlainStringVarValidation(t *testing.T) {
	prevCfg := Cfg
	Cfg = &BeastConfig{
		AllowedBaseImages: []string{core.DEFAULT_BASE_IMAGE},
	}
	defer func() { Cfg = prevCfg }()

	dir := t.TempDir()
	env := ChallengeEnv{
		Ports: []uint32{8080},
		EnvironmentVars: []EnvironmentVar{
			{Key: "FOO", Value: "hello_world"},
			{Key: "EMPTY_VAL_ALLOWED", Value: ""},
		},
	}
	if err := env.ValidateRequiredFields(core.SERVICE_CHALLENGE_TYPE_NAME, dir); err != nil {
		t.Fatalf("expected plain-string EnvironmentVars to validate, got: %v", err)
	}

	invalidEnv := ChallengeEnv{
		Ports: []uint32{8080},
		EnvironmentVars: []EnvironmentVar{
			{Key: "", Value: "hello_world"},
		},
	}
	if err := invalidEnv.ValidateRequiredFields(core.SERVICE_CHALLENGE_TYPE_NAME, dir); err == nil {
		t.Fatalf("expected empty env key to fail validation")
	}
}

func TestOpenSSHServerInjectionScopedToInstancedSSH(t *testing.T) {
	prevCfg := Cfg
	Cfg = &BeastConfig{
		AllowedBaseImages: []string{core.DEFAULT_BASE_IMAGE},
	}
	defer func() { Cfg = prevCfg }()

	dir := t.TempDir()

	// 1. Non-instanced service challenge should NOT get openssh-server injected
	nonInstanced := Challenge{
		Metadata: ChallengeMetadata{
			Name:       "svc-plain",
			Flag:       "CTF{ok}",
			Type:       core.SERVICE_CHALLENGE_TYPE_NAME,
			Difficulty: "easy",
		},
		Env: ChallengeEnv{
			Ports: []uint32{9000},
		},
	}
	if err := nonInstanced.ValidateRequiredFields(dir); err != nil {
		t.Fatalf("non-instanced service validate: %v", err)
	}
	if slices.Contains(nonInstanced.Env.AptDeps, "openssh-server") {
		t.Fatalf("did not expect openssh-server on non-instanced challenge, got %v", nonInstanced.Env.AptDeps)
	}

	// 2. Instanced non-SSH service challenge (port 9000, no port 22) should NOT get openssh-server injected
	instancedNonSSH := Challenge{
		Metadata: ChallengeMetadata{
			Name:       "svc-inst-nonssh",
			Flag:       "CTF{ok}",
			Type:       core.SERVICE_CHALLENGE_TYPE_NAME,
			Difficulty: "easy",
			Instanced:  true,
		},
		Env: ChallengeEnv{
			Ports: []uint32{9000},
		},
	}
	if err := instancedNonSSH.ValidateRequiredFields(dir); err != nil {
		t.Fatalf("instanced non-ssh validate: %v", err)
	}
	if slices.Contains(instancedNonSSH.Env.AptDeps, "openssh-server") {
		t.Fatalf("did not expect openssh-server on instanced non-SSH challenge, got %v", instancedNonSSH.Env.AptDeps)
	}

	// 3. Instanced SSH challenge (port 22) SHOULD get openssh-server injected
	instancedSSH := Challenge{
		Metadata: ChallengeMetadata{
			Name:       "svc-inst-ssh",
			Flag:       "CTF{ok}",
			Type:       core.SERVICE_CHALLENGE_TYPE_NAME,
			Difficulty: "easy",
			Instanced:  true,
		},
		Env: ChallengeEnv{
			Ports: []uint32{core.SSH_PORT},
		},
	}
	if err := instancedSSH.ValidateRequiredFields(dir); err != nil {
		t.Fatalf("instanced ssh validate: %v", err)
	}
	if !slices.Contains(instancedSSH.Env.AptDeps, "openssh-server") {
		t.Fatalf("expected openssh-server injected for instanced SSH challenge, got %v", instancedSSH.Env.AptDeps)
	}
}

func TestInstanceConfigValidatePreservesCustomDefaultExpiration(t *testing.T) {
	ic := InstanceConfig{
		DefaultExpiration: 900,
		MaxExtension:      3600,
	}
	ic.Validate()
	if ic.DefaultExpiration != 900 || ic.MaxExtension != 3600 {
		t.Fatalf("expected DefaultExpiration=900 and MaxExtension=3600 preserved, got %+v", ic)
	}

	icTooSmallMax := InstanceConfig{
		DefaultExpiration: 1800,
		MaxExtension:      600,
	}
	icTooSmallMax.Validate()
	if icTooSmallMax.DefaultExpiration != 1800 || icTooSmallMax.MaxExtension != 1800 {
		t.Fatalf("expected MaxExtension raised to DefaultExpiration (1800), got %+v", icTooSmallMax)
	}
}

func TestValidateConfigUnauthenticatedMailNoFalseWarning(t *testing.T) {
	tmpDir := t.TempDir()
	authKeys := filepath.Join(tmpDir, "authorized_keys")
	scriptsDir := filepath.Join(tmpDir, "scripts")

	var buf bytes.Buffer
	prevOut := log.StandardLogger().Out
	prevLevel := log.GetLevel()
	log.SetOutput(&buf)
	log.SetLevel(log.WarnLevel)
	defer func() {
		log.SetOutput(prevOut)
		log.SetLevel(prevLevel)
	}()

	cfg := BeastConfig{
		AuthorizedKeysFile: authKeys,
		BeastScriptsDir:    scriptsDir,
		JWTSecret:          "test-jwt-secret",
		BeastStaticUrl:     "http://localhost:8080",
		PsqlConf: PsqlConfig{
			User:     "beast",
			Password: "pw",
			Dbname:   "beast",
			Host:     "localhost",
			Port:     "5432",
			SslMode:  "disable",
		},
		RedisConf: RedisConfig{
			Host: "localhost",
			Port: "6379",
		},
		MailConfig: MailConfig{
			From:     "noreply@example.test",
			Password: "",
			SMTPHost: "localhost",
			SMTPPort: "51025",
		},
	}

	if err := cfg.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig failed: %v", err)
	}
	if strings.Contains(buf.String(), "Mail configuration not provided") {
		t.Fatalf("unexpected mail warning with From/SMTPHost/SMTPPort set: %s", buf.String())
	}
}
