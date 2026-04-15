package remoteManager

import (
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
	"strings"
)

func CleanupOrphanedOnServer(serverDeployed string) {
	var containers []types.Container
	var err error

	server := config.Cfg.AvailableServers[serverDeployed]
	containers, err = SearchContainerByFilterRemote(map[string]string{
		"label": "beast.instance=true",
	}, server)

	if err != nil {
		log.Warnf("Failed to search for instance containers on %s: %v", serverDeployed, err)
		return
	}

	for _, container := range containers {
		instanceID := container.Labels["beast.instance.id"]
		if instanceID == "" {
			for _, name := range container.Names {
				name = strings.TrimPrefix(name, "/")
				if strings.HasPrefix(name, "beast_instance_") {
					parts := strings.Split(name, "_")
					if len(parts) >= 4 {
						instanceID = parts[len(parts)-1]
						break
					}
				}
			}
		}

		if instanceID == "" {
			continue
		}

		_, err = cache.GetInstance(instanceID)
		if err != nil {
			containerName := ""
			if len(container.Names) > 0 {
				containerName = strings.TrimPrefix(container.Names[0], "/")
			}
			log.Infof("Removing orphaned instance container: %s (instance %s) on %s", containerName, instanceID, serverDeployed)

			err = StopAndRemoveContainerRemote(container.ID, server)
			if err != nil {
				log.Warnf("Failed to remove orphaned container %s on %s: %v", container.ID[:12], serverDeployed, err)
			}

			err = cache.FreeContainerPortsOnHost(serverDeployed, container.ID)
			if err != nil {
				log.Warnf("Failed to free port for orphan container %s on server %s: %s", container.ID[:12], serverDeployed, err.Error())
			}
		}
	}
}

// cleanupOrphanedComposeInstancesOnServer finds and removes orphaned docker compose instance projects.
// Instanced stacks use ComposeDockerProjectNameInstanced; compose ls names use prefix "beast-instance-".
func CleanupOrphanedComposeInstancesOnServer(serverDeployed string) {
	var projectNames []string
	var err error

	server := config.Cfg.AvailableServers[serverDeployed]
	projectNames, err = getOrphanedComposeInstanceProjectsRemote(server)

	if err != nil {
		log.Warnf("Failed to get compose instance projects on %s: %v", serverDeployed, err)
		return
	}

	for _, projectName := range projectNames {
		// Extract instance ID from project name: beast-instance-{encoded_challenge}-{instanceID}
		parts := strings.Split(projectName, "-")
		if len(parts) < 4 {
			continue
		}
		instanceID := parts[len(parts)-1]

		// Check if instance still exists in cache
		_, err := cache.GetInstance(instanceID)
		if err != nil {
			log.Infof("Removing orphaned compose instance project: %s (instance %s) on %s", projectName, instanceID, serverDeployed)

			if err := composeDownProjectRemote(projectName, server); err != nil {
				log.Warnf("Failed to remove orphaned compose project %s on %s: %v", projectName, serverDeployed, err)
			}
		}
	}
}

// getOrphanedComposeInstanceProjectsRemote returns compose instance projects on a remote server
func getOrphanedComposeInstanceProjectsRemote(server config.AvailableServer) ([]string, error) {
	output, err := RunCommandOnServer(server, "docker compose ls --format json")
	if err != nil {
		return nil, fmt.Errorf("docker compose ls failed on remote: %v", err)
	}

	type ComposeProject struct {
		Name   string `json:"Name"`
		Status string `json:"Status"`
	}

	var projects []ComposeProject
	outputStr := strings.TrimSpace(output)
	if outputStr == "" {
		return nil, nil
	}

	if err := json.Unmarshal([]byte(outputStr), &projects); err != nil {
		// Try parsing line by line
		for _, line := range strings.Split(outputStr, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var project ComposeProject
			if err := json.Unmarshal([]byte(line), &project); err != nil {
				continue
			}
			projects = append(projects, project)
		}
	}

	var instanceProjects []string
	for _, project := range projects {
		if strings.HasPrefix(project.Name, "beast-instance-") {
			instanceProjects = append(instanceProjects, project.Name)
		}
	}

	return instanceProjects, nil
}

// composeDownProjectRemote removes a docker compose project on a remote server
func composeDownProjectRemote(projectName string, server config.AvailableServer) error {
	cmd := fmt.Sprintf("docker compose -p %s down --remove-orphans -v", projectName)
	output, err := RunCommandOnServer(server, cmd)
	if err != nil {
		return fmt.Errorf("docker compose down failed on remote: %v, output: %s", err, output)
	}

	log.Debugf("Successfully removed compose project %s on %s", projectName, server.Host)
	return nil
}
