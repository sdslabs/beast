package utils

import (
	"github.com/sdslabs/beastv4/core/cache"
	log "github.com/sirupsen/logrus"
)

func AssignPortsOnContainerToHost(serverDeployed string, containerID string, ports []uint32) error {
	/* Failure should be treated as fatal since this can lead to a leak... */
	if err := cache.AssignPortsOnHostToContainer(serverDeployed, containerID, ports); err != nil {
		log.Warnf("Failed to register ports %v for container %s: %v", ports, containerID, err)
		return err
	}

	return nil
}

func FreePortsOnHost(serverDeployed string, ports []uint32) {
	for _, port := range ports {
		if err := cache.FreePortOnHost(serverDeployed, port); err != nil {
			log.Warnf("Failed to free port %d for host %s: %v", port, serverDeployed, err)
		}
	}
}

func AssignPortsOnContainerToHostCompose(serverDeployed string, containerID string, ports map[string]uint32) error {
	portList := make([]uint32, 0, len(ports))
	for _, port := range ports {
		portList = append(portList, port)
	}

	/* Failure should be treated as fatal since this can lead to a leak... */
	if err := cache.AssignPortsOnHostToContainer(serverDeployed, containerID, portList); err != nil {
		log.Warnf("Failed to register ports %v for container %s: %v", portList, containerID, err)
		return err
	}

	return nil
}

func FreePortsOnHostCompose(serverDeployed string, ports map[string]uint32) {
	for _, port := range ports {
		if err := cache.FreePortOnHost(serverDeployed, port); err != nil {
			log.Warnf("Failed to free port %d for host %s: %v", port, serverDeployed, err)
		}
	}
}
