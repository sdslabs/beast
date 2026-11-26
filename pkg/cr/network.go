package cr

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

func VerifyHydraNetwork(networkName, bridgeName string) error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}

	network, err := cli.NetworkInspect(context.Background(), networkName, types.NetworkInspectOptions{})
	if err != nil {
		return fmt.Errorf("required Docker network %q is missing or inaccessible; run scripts/provision/hydra-net-setup.sh on this host: %w", networkName, err)
	}
	if network.Driver != "bridge" {
		return fmt.Errorf("Docker network %q must use bridge driver, got %q", networkName, network.Driver)
	}
	if got := network.Options["com.docker.network.bridge.name"]; got != bridgeName {
		return fmt.Errorf("Docker network %q must use bridge interface %q, got %q", networkName, bridgeName, got)
	}

	return nil
}

func GetSingleInternalNetworkForContainer(containerID, hydraNetworkName string) (string, error) {
	cli, err := newDockerClient()
	if err != nil {
		return "", err
	}

	inspect, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return "", err
	}

	internalNetworks := make([]string, 0)
	for networkName := range inspect.NetworkSettings.Networks {
		if networkName == hydraNetworkName {
			continue
		}

		network, err := cli.NetworkInspect(context.Background(), networkName, types.NetworkInspectOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to inspect network %q for container %s: %w", networkName, containerID, err)
		}
		if network.Internal {
			internalNetworks = append(internalNetworks, networkName)
		}
	}

	if len(internalNetworks) != 1 {
		return "", fmt.Errorf("container %s must be attached to exactly one internal checker network, got %v", containerID, internalNetworks)
	}
	return internalNetworks[0], nil
}

func GetNamedVolumeForContainerMount(containerID, mountTarget string) (string, error) {
	cli, err := newDockerClient()
	if err != nil {
		return "", err
	}

	inspect, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return "", err
	}

	for _, mnt := range inspect.Mounts {
		if mnt.Destination != mountTarget {
			continue
		}
		if string(mnt.Type) != "volume" {
			return "", fmt.Errorf("mount %s in container %s must be a named volume, got %s", mountTarget, containerID, mnt.Type)
		}
		if mnt.Name == "" {
			return "", fmt.Errorf("mount %s in container %s has no volume name", mountTarget, containerID)
		}
		return mnt.Name, nil
	}

	return "", fmt.Errorf("container %s does not have a named volume mounted at %s", containerID, mountTarget)
}
