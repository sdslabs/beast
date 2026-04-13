package utils

import (
	"github.com/sdslabs/beastv4/core/cache"
	log "github.com/sirupsen/logrus"
)

func AssignPortsOnContainerToHost(serverDeployed string, containerID string, ports []uint32) {
	for _, port := range ports {
		/* Failure should be treated as fatal since this can lead to a leak... */
		if err := cache.AssignFreePortOnHostToContainer(serverDeployed, containerID, port); err != nil {
			log.Warnf("Failed to register port %d for container %s: %v", port, containerID, err)
		}
	}
}

func FreePortsOnHost(serverDeployed string, ports []uint32) {
	for _, port := range ports {
		if err := cache.FreePortOnHost(serverDeployed, port); err != nil {
			log.Warnf("Failed to free port %d for host %s: %v", port, serverDeployed, err)
		}
	}
}

func AssignPortsOnContainerToHostCompose(serverDeployed string, containerID string, ports map[string]uint32) {
	for _, port := range ports {
		/* Failure should be treated as fatal since this can lead to a leak... */
		if err := cache.AssignFreePortOnHostToContainer(serverDeployed, containerID, port); err != nil {
			log.Warnf("Failed to register port %d for container %s: %v", port, containerID, err)
		}
	}
}

func FreePortsOnHostCompose(serverDeployed string, ports map[string]uint32) {
	for _, port := range ports {
		if err := cache.FreePortOnHost(serverDeployed, port); err != nil {
			log.Warnf("Failed to free port %d for host %s: %v", port, serverDeployed, err)
		}
	}
}
