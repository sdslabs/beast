package remoteManager

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sdslabs/beastv4/core/config"
)

func VerifyHydraNetworkRemote(server config.AvailableServer, networkName, bridgeName string) error {
	command := fmt.Sprintf(
		"docker network inspect %s --format '{{.Driver}} {{index .Options \"com.docker.network.bridge.name\"}}'",
		shellQuote(networkName),
	)
	output, err := RunCommandOnServer(server, command)
	if err != nil {
		return fmt.Errorf("required Docker network %q is missing or inaccessible on host %s; run scripts/provision/hydra-net-setup.sh on that host: %w", networkName, server.Host, err)
	}

	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 2 {
		return fmt.Errorf("could not inspect Docker network %q on host %s", networkName, server.Host)
	}
	if fields[0] != "bridge" {
		return fmt.Errorf("Docker network %q on host %s must use bridge driver, got %q", networkName, server.Host, fields[0])
	}
	if fields[1] != bridgeName {
		return fmt.Errorf("Docker network %q on host %s must use bridge interface %q, got %q", networkName, server.Host, bridgeName, fields[1])
	}

	return nil
}

func GetSingleInternalNetworkForContainerRemote(server config.AvailableServer, containerID, hydraNetworkName string) (string, error) {
	command := fmt.Sprintf(
		"docker inspect %s --format '{{range $name, $_ := .NetworkSettings.Networks}}{{println $name}}{{end}}'",
		shellQuote(containerID),
	)
	output, err := RunCommandOnServer(server, command)
	if err != nil {
		return "", err
	}

	internalNetworks := make([]string, 0)
	for _, networkName := range strings.Fields(output) {
		if networkName == hydraNetworkName {
			continue
		}
		inspectCommand := fmt.Sprintf(
			"docker network inspect %s --format '{{.Internal}}'",
			shellQuote(networkName),
		)
		inspectOutput, err := RunCommandOnServer(server, inspectCommand)
		if err != nil {
			return "", fmt.Errorf("failed to inspect network %q for container %s on host %s: %w", networkName, containerID, server.Host, err)
		}
		if strings.TrimSpace(inspectOutput) == "true" {
			internalNetworks = append(internalNetworks, networkName)
		}
	}

	if len(internalNetworks) != 1 {
		return "", fmt.Errorf("container %s on host %s must be attached to exactly one internal checker network, got %v", containerID, server.Host, internalNetworks)
	}
	return internalNetworks[0], nil
}

func GetNamedVolumeForContainerMountRemote(server config.AvailableServer, containerID, mountTarget string) (string, error) {
	command := fmt.Sprintf(
		"docker inspect %s --format '{{range .Mounts}}{{if eq .Destination %s}}{{println .Type .Name}}{{end}}{{end}}'",
		shellQuote(containerID),
		strconv.Quote(mountTarget),
	)
	output, err := RunCommandOnServer(server, command)
	if err != nil {
		return "", err
	}

	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 2 {
		return "", fmt.Errorf("container %s on host %s does not have a named volume mounted at %s", containerID, server.Host, mountTarget)
	}
	if fields[0] != "volume" {
		return "", fmt.Errorf("mount %s in container %s on host %s must be a named volume, got %s", mountTarget, containerID, server.Host, fields[0])
	}
	return fields[1], nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
