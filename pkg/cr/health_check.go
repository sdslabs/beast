package cr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
)

func CleanupOrphans() {
	var containers []types.Container
	var err error

	containers, err = SearchContainerByFilter(map[string]string{
		"label": "beast.instance=true",
	})

	if err != nil {
		log.Warnf("Failed to search for instance containers on %s: %v", core.LOCALHOST, err)
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
			log.Infof("Removing orphaned instance container: %s (instance %s) on %s", containerName, instanceID, core.LOCALHOST)

			err = StopAndRemoveContainer(container.ID)
			if err != nil {
				log.Warnf("Failed to remove orphaned container %s: %s", container.ID[:12], err.Error())
			}

			err = cache.FreeContainerPortsOnHost(core.LOCALHOST, container.ID)
			if err != nil {
				log.Warnf("Failed to free port for orphan container %s: %s", container.ID[:12], err.Error())
			}
		}
	}
}

// CleanupOrphanedComposeInstances finds and removes orphaned docker compose instance projects.
// Instanced compose uses ComposeDockerProjectNameInstanced (-p = beast-instance-<encoded>-<instanceId>);
// compose ls project names are matched by prefix "beast-instance-".
func CleanupOrphanedComposeInstances() {
	var projectNames []string
	var err error

	projectNames, err = getOrphanedComposeInstanceProjects()

	if err != nil {
		log.Warnf("Failed to get compose instance projects on %s: %v", core.LOCALHOST, err)
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
		_, err = cache.GetInstance(instanceID)
		if err != nil {
			log.Infof("Removing orphaned compose instance project: %s (instance %s) on %s", projectName, instanceID, core.LOCALHOST)

			err = composeDownProject(projectName)
			if err != nil {
				log.Warnf("Failed to remove orphaned compose project %s: %v", projectName, err)
			}
		}
	}
}

// getOrphanedComposeInstanceProjects returns a list of docker compose project names
// that match the instance naming pattern (beast-instance-*)
func getOrphanedComposeInstanceProjects() ([]string, error) {
	cmd := exec.Command("docker", "compose", "ls", "--format", "json")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker compose ls failed: %v, output: %s", err, output.String())
	}

	type ComposeProject struct {
		Name   string `json:"Name"`
		Status string `json:"Status"`
	}

	var projects []ComposeProject
	outputStr := strings.TrimSpace(output.String())
	if outputStr == "" {
		return nil, nil
	}

	if err := json.Unmarshal([]byte(outputStr), &projects); err != nil {
		// Try parsing line by line (older docker compose versions)
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

// composeDownProject removes a docker compose project by name
func composeDownProject(projectName string) error {
	cmd := exec.Command("docker", "compose", "-p", projectName, "down", "--remove-orphans", "-v")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose down failed: %v, output: %s", err, output.String())
	}

	log.Debugf("Successfully removed compose project %s", projectName)
	return nil
}
