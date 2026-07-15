package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCompose(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "compose.yml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractPortsFromComposeRejectsHostEscapeSettings(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		want    string
	}{
		{name: "privileged", setting: "    privileged: true\n", want: "cannot be privileged"},
		{name: "host network", setting: "    network_mode: host\n", want: "forbidden network_mode"},
		{name: "host pid", setting: "    pid: host\n", want: "host namespace"},
		{name: "device", setting: "    devices: [/dev/kvm:/dev/kvm]\n", want: "additional host privileges"},
		{name: "capability", setting: "    cap_add: [SYS_ADMIN]\n", want: "additional host privileges"},
		{name: "runtime", setting: "    runtime: runc\n", want: "host-managed container settings"},
		{name: "socket", setting: "    volumes: [/var/run/docker.sock:/var/run/docker.sock]\n", want: "host volume source"},
		{name: "build escape", setting: "    build: ..\n", want: "invalid build context"},
		{name: "env escape", setting: "    env_file: ../secret.env\n", want: "invalid env_file"},
		{name: "volumes from", setting: "    volumes_from: [host-container]\n", want: "host-managed services"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeCompose(t, "services:\n  app:\n"+test.setting+"    ports: [\"${APP_PORT}:80\"]\n")
			_, err := ExtractPortsFromCompose(path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestExtractPortsFromComposeAllowsProjectBindMount(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "init.sql"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "compose.yml")
	content := "services:\n  app:\n    volumes: [\"./init.sql:/init.sql:ro\"]\n    ports: [\"${APP_PORT}:80\"]\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	ports, err := ExtractPortsFromCompose(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ports) != 1 || ports[0] != "APP_PORT" {
		t.Fatalf("unexpected ports: %v", ports)
	}
}

func TestExtractPortsFromComposeRejectsEscapingBindMount(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "challenge")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "secret"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "compose.yml")
	content := "services:\n  app:\n    volumes: [\"../secret:/secret:ro\"]\n    ports: [\"${APP_PORT}:80\"]\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ExtractPortsFromCompose(path)
	if err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("expected escaping bind mount error, got %v", err)
	}
}
