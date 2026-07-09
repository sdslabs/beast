package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeComposeTestFile(t *testing.T, content string) string {
	t.Helper()
	return writeComposeTestFileInDir(t, t.TempDir(), content)
}

func writeComposeTestFileInDir(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	return path
}

func writeCheckScript(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "check.sh"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write check.sh: %v", err)
	}
}

func TestValidateInstancedComposeSSHContractValid(t *testing.T) {
	composeFile := writeComposeTestFile(t, `
version: '3.8'
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks:
      - exposed
      - internal
  db:
    image: mysql:5.7
    networks:
      - internal
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    driver: bridge
    internal: true
volumes:
  challenge:
`)

	if err := ValidateInstancedComposeSSHContract(composeFile, "SSH_PORT", "hydra-net", "ssh", 22); err != nil {
		t.Fatalf("expected valid compose contract, got %v", err)
	}
}

func TestValidateInstancedComposeSSHContractRejectsBackendOnHydra(t *testing.T) {
	composeFile := writeComposeTestFile(t, `
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks: [exposed, internal]
  db:
    image: mysql:5.7
    networks: [exposed, internal]
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    internal: true
volumes:
  challenge:
`)

	err := ValidateInstancedComposeSSHContract(composeFile, "SSH_PORT", "hydra-net", "ssh", 22)
	if err == nil || !strings.Contains(err.Error(), "must not attach to external network") {
		t.Fatalf("expected backend hydra-net rejection, got %v", err)
	}
}

func TestValidateInstancedComposeSSHContractRejectsWrongDefaultPort(t *testing.T) {
	composeFile := writeComposeTestFile(t, `
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks: [exposed, internal]
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    internal: true
volumes:
  challenge:
`)

	err := ValidateInstancedComposeSSHContract(composeFile, "INSTANCE_PORT", "hydra-net", "ssh", 22)
	if err == nil || !strings.Contains(err.Error(), "default_port_var") {
		t.Fatalf("expected default_port_var rejection, got %v", err)
	}
}

func TestValidateInstancedComposeSSHContractRejectsDangerousNetworkConfig(t *testing.T) {
	composeFile := writeComposeTestFile(t, `
services:
  ssh:
    image: ubuntu:24.04
    privileged: true
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks: [exposed, internal]
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    internal: true
volumes:
  challenge:
`)

	err := ValidateInstancedComposeSSHContract(composeFile, "SSH_PORT", "hydra-net", "ssh", 22)
	if err == nil || !strings.Contains(err.Error(), "privileged") {
		t.Fatalf("expected privileged rejection, got %v", err)
	}
}

func TestExtractPortsFromComposeSupportsLongSyntax(t *testing.T) {
	composeFile := writeComposeTestFile(t, `
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - target: 22
        published: "${SSH_PORT}"
        protocol: tcp
`)

	ports, err := ExtractPortsFromCompose(composeFile)
	if err != nil {
		t.Fatalf("extract ports: %v", err)
	}
	if len(ports) != 1 || ports[0] != "SSH_PORT" {
		t.Fatalf("expected SSH_PORT, got %#v", ports)
	}
}

func TestValidateSadServersComposeContractRejectsUnsafeOptions(t *testing.T) {
	tests := []struct {
		name    string
		service string
		want    string
	}{
		{
			name:    "docker socket bind",
			service: `volumes: ["challenge:/challenge", "/var/run/docker.sock:/var/run/docker.sock"]`,
			want:    "Docker socket",
		},
		{
			name:    "host bind",
			service: `volumes: ["challenge:/challenge", "./data:/data"]`,
			want:    "host bind mount",
		},
		{
			name:    "privileged",
			service: `privileged: true`,
			want:    "privileged",
		},
		{
			name:    "pid host",
			service: `pid: host`,
			want:    "pid: host",
		},
		{
			name:    "non empty network mode",
			service: `network_mode: bridge`,
			want:    "network_mode",
		},
		{
			name:    "user namespace mode",
			service: `userns_mode: host`,
			want:    "userns_mode",
		},
		{
			name:    "cgroup namespace mode",
			service: `cgroupns_mode: host`,
			want:    "cgroupns_mode",
		},
		{
			name:    "cgroup parent",
			service: `cgroup_parent: beast.slice`,
			want:    "cgroup_parent",
		},
		{
			name:    "extra hosts",
			service: `extra_hosts: ["host.docker.internal:host-gateway"]`,
			want:    "extra_hosts",
		},
		{
			name:    "volumes from",
			service: `volumes_from: ["ssh"]`,
			want:    "volumes_from",
		},
		{
			name:    "env file",
			service: `env_file: [".env"]`,
			want:    "env_file",
		},
		{
			name:    "secrets",
			service: `secrets: ["db_password"]`,
			want:    "secrets",
		},
		{
			name:    "configs",
			service: `configs: ["app_config"]`,
			want:    "configs",
		},
		{
			name: "sysctls",
			service: `sysctls:
      net.ipv4.ip_forward: "1"`,
			want: "sysctls",
		},
		{
			name: "ulimits",
			service: `ulimits:
      nofile: 65535`,
			want: "ulimits",
		},
		{
			name:    "cap add",
			service: `cap_add: ["SYS_ADMIN"]`,
			want:    "capabilities",
		},
		{
			name:    "sidecar port",
			service: `ports: ["${DB_PORT}:3306"]`,
			want:    "must not publish host ports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCheckScript(t, dir)
			composeFile := writeComposeTestFileInDir(t, dir, `
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks: [exposed, internal]
  db:
    image: mysql:5.7
    networks: [internal]
    `+tt.service+`
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    internal: true
volumes:
  challenge:
`)

			err := ValidateInstancedComposeSSHContract(composeFile, "SSH_PORT", "hydra-net", "ssh", 22, true)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q rejection, got %v", tt.want, err)
			}
		})
	}
}

func TestValidateSadServersComposeContractRejectsExtraExternalNetwork(t *testing.T) {
	dir := t.TempDir()
	writeCheckScript(t, dir)
	composeFile := writeComposeTestFileInDir(t, dir, `
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks: [exposed, internal]
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    internal: true
  internet:
    external: true
    name: internet
volumes:
  challenge:
`)

	err := ValidateInstancedComposeSSHContract(composeFile, "SSH_PORT", "hydra-net", "ssh", 22, true)
	if err == nil || !strings.Contains(err.Error(), "extra network") {
		t.Fatalf("expected extra network rejection, got %v", err)
	}
}

func TestValidateSadServersComposeContractRejectsExtraInternalNetwork(t *testing.T) {
	dir := t.TempDir()
	writeCheckScript(t, dir)
	composeFile := writeComposeTestFileInDir(t, dir, `
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks: [exposed, internal]
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    internal: true
  debug:
    internal: true
volumes:
  challenge:
`)

	err := ValidateInstancedComposeSSHContract(composeFile, "SSH_PORT", "hydra-net", "ssh", 22, true)
	if err == nil || !strings.Contains(err.Error(), "exactly one non-exposed internal network") {
		t.Fatalf("expected extra internal network rejection, got %v", err)
	}
}

func TestValidateSadServersComposeContractRejectsServiceNetworkOptions(t *testing.T) {
	dir := t.TempDir()
	writeCheckScript(t, dir)
	composeFile := writeComposeTestFileInDir(t, dir, `
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks:
      exposed:
      internal:
        aliases:
          - shell
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    internal: true
volumes:
  challenge:
`)

	err := ValidateInstancedComposeSSHContract(composeFile, "SSH_PORT", "hydra-net", "ssh", 22, true)
	if err == nil || !strings.Contains(err.Error(), "network options") {
		t.Fatalf("expected per-service network options rejection, got %v", err)
	}
}

func TestValidateSadServersComposeContractRequiresCheckScript(t *testing.T) {
	dir := t.TempDir()
	composeFile := writeComposeTestFileInDir(t, dir, `
services:
  ssh:
    image: ubuntu:24.04
    ports:
      - "${SSH_PORT}:22"
    volumes:
      - challenge:/challenge
    networks: [exposed, internal]
networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    internal: true
volumes:
  challenge:
`)

	err := ValidateInstancedComposeSSHContract(composeFile, "SSH_PORT", "hydra-net", "ssh", 22, true)
	if err == nil || !strings.Contains(err.Error(), "check.sh") {
		t.Fatalf("expected check.sh rejection, got %v", err)
	}
}
