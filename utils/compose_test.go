package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeComposeTestFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "docker-compose.yml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	return path
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
