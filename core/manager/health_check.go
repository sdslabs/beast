package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/notify"
	"github.com/sdslabs/beastv4/pkg/probes"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

var HEALTH_CHECKER = false

// Check for static challenegs' assets to be present on staging server.
// At the time of writing, Beast deploys assets to localhost only.
// So it will check only on localhost
func CheckStaticChallenge(chall database.Challenge) error {
	if chall.Assets == "" {
		return nil
	}

	assets := strings.Split(chall.Assets, core.DELIMITER)
	for _, asset := range assets {
		filepath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, chall.Name, core.BEAST_STATIC_FOLDER, asset)
		err := utils.ValidateFileExists(filepath)
		if err != nil {
			err = fmt.Errorf("static chall: %s not staged. Asset: %s Missing", chall.Name, asset)
			log.Error(err)
			return err
		}
	}
	return nil
}

// Check for container running or not.
func containerProber(chall database.Challenge) error {
	challHost := chall.ServerDeployed
	if challHost == core.LOCALHOST || challHost == "" {
		containers, err := cr.SearchRunningContainerByFilter(map[string]string{"id": chall.ContainerId})
		if err != nil || len(containers) <= 0 {
			err = fmt.Errorf("error while searching for container with id %s on server: %s", chall.ContainerId, chall.ServerDeployed)
			return err
		}
	} else {
		server := config.Cfg.AvailableServers[chall.ServerDeployed]
		containers, err := remoteManager.SearchRunningContainerByFilterRemote(map[string]string{"id": chall.ContainerId}, server)
		if err != nil || len(containers) <= 0 {
			err = fmt.Errorf("error while searching for container with id %s on remote server: %s", chall.ContainerId, chall.ServerDeployed)
			return err
		}
	}
	return nil
}

// Check for challenge running or not
func ChallengesHealthProber(waitTime int) {
	log.Info("Starting Challenge Health Check prober.")
	challs, err := database.QueryChallengeEntriesMap(map[string]interface{}{
		"status":       core.DEPLOY_STATUS["deployed"],
		"health_check": 1,
	})

	if err != nil {
		log.Errorf("Error while querying challenges : %v", err)
		return
	}

	for _, chall := range challs {
		if chall.Format != core.STATIC_CHALLENGE_TYPE_NAME {
			allocatedPorts, err := database.GetAllocatedPorts(chall)
			if err != nil {
				log.Errorf("Error while accessing database : %v", err)
				continue
			}

			log.Debugf("Doing HealthCheck Probe for %s", chall.Name)

			// Do a better job at health probing mechanism.
			if len(allocatedPorts) > 0 {
				port := int(allocatedPorts[0].PortNo)
				prober := probes.NewTcpProber()
				result, err := prober.Probe(chall.ServerDeployed, port, time.Duration(core.DEFAULT_PROBE_TIMEOUT)*time.Second)
				if err != nil {
					msg := fmt.Sprintf("NETWORK HEALTH CHECK %s: %s : %s", result, chall.Name, err)
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Error(msg)
					go notify.SendNotification(notify.Error, msg)
				} else {
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Info("NETWORK HEALTH CHECK returned success.")
				}
				err = containerProber(chall)
				if err != nil {
					msg := fmt.Sprintf("CONTAINER HEALTH CHECK %s: %s : %s", result, chall.Name, err)
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Error(msg)
					go notify.SendNotification(notify.Error, msg)
				} else {
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Info("CONTAINER HEALTH CHECK returned success.")
				}
			}
		} else {
			err := CheckStaticChallenge(chall)
			if err != nil {
				msg := fmt.Sprintf("HEALTHCHECK Failure: %s : %s", chall.Name, err)
				log.WithFields(log.Fields{
					"ChallName": chall.Name,
				}).Error(msg)
				go notify.SendNotification(notify.Error, msg)
			}
		}
	}
}

// Check for Remote Server running or not
func ServerHealthProber(waitTime int) {
	for _, server := range config.Cfg.AvailableServers {
		if server.Active && server.Host != core.LOCALHOST {
			err := remoteManager.PingServer(server)
			if err != nil {
				msg := fmt.Sprintf("SERVER HEALTH CHECK Faliure: %s : %s", server.Host, err)
				log.WithFields(log.Fields{
					"ChallName": server.Host,
				}).Error(msg)
				go notify.SendNotification(notify.Error, msg)
			} else {
				log.WithFields(log.Fields{
					"ChallName": server.Host,
				}).Info("SERVER HEALTH CHECK returned success.")
			}
		}
	}
}

func BeastHeathCheckProber(waitTime int) {
	if !HEALTH_CHECKER {
		log.Info("Starting Health Check prober.")
		HEALTH_CHECKER = true

		go InstanceCleanupProber()

		for {
			go ChallengesHealthProber(waitTime)
			go ServerHealthProber(waitTime)
			go database.BackupDatabase()
			go cache.BackupCache()
			time.Sleep(time.Duration(waitTime) * time.Second)
		}
	} else {
		log.Warn("Health Checker Already Running. Not Starting Again")
	}
}

func InstanceCleanupProber() {
	log.Info("Starting Instance Cleanup prober with interval: ", core.DEFAULT_HEALTH_CHECK_TIME)

	for {
		QueueExpiredInstances()
		ProcessInstanceDeletionQueue()
		CleanupOrphanedInstanceContainers()
		time.Sleep(core.DEFAULT_HEALTH_CHECK_TIME)
	}
}

func QueueExpiredInstances() {
	log.Debug("Checking for expired instances")

	expired, err := cache.GetExpiredInstances()
	if err != nil {
		log.Warnf("Failed to get expired instances: %v", err)
		return
	}

	for _, instance := range expired {
		log.Infof("Instance %s expired (challenge: %s, user: %s), queueing for deletion",
			instance.InstanceID, instance.ChallengeName, instance.UserID)

		err := cache.QueueInstanceForDeletion(instance.InstanceID)
		if err != nil {
			log.Warnf("Failed to queue instance %s for deletion: %v", instance.InstanceID, err)
		}
	}
}

func ProcessInstanceDeletionQueue() {
	log.Debug("Processing instance deletion queue")

	for i := 0; i < 10; i++ {
		instance, err := cache.PopInstanceForDeletion()
		if err != nil {
			log.Warnf("Error popping from deletion queue: %v", err)
			return
		}

		if instance == nil {
			return
		}

		log.Infof("Processing deletion for instance %s (challenge: %s, container: %s, server: %s)",
			instance.InstanceID, instance.ChallengeName, instance.ContainerID, instance.ServerDeployed)

		err = killInstanceContainer(instance.ContainerID, instance.DeploymentType, instance.InstanceID, instance.ChallengeName, instance.ServerDeployed)
		if err != nil {
			log.Warnf("Failed to kill container for instance %s: %v", instance.InstanceID, err)
		} else {
			log.Infof("Successfully killed container for instance %s", instance.InstanceID)
		}

		cache.FreeContainerPortsOnHost(instance.ServerDeployed, instance.ContainerID)
	}

	queueLen, _ := cache.GetDeletionQueueLength()
	if queueLen > 0 {
		log.Debugf("Deletion queue still has %d items, will process in next cycle", queueLen)
	}
}

func CleanupOrphanedInstanceContainers() {
	log.Debug("Checking for orphaned instance containers")

	cleanupOrphanedOnServer(core.LOCALHOST)
	cleanupOrphanedComposeInstancesOnServer(core.LOCALHOST)

	for host, server := range config.Cfg.AvailableServers {
		if server.Active && host != core.LOCALHOST {
			cleanupOrphanedOnServer(host)
			cleanupOrphanedComposeInstancesOnServer(host)
		}
	}
}

func cleanupOrphanedOnServer(serverHost string) {
	var containers []types.Container
	var err error

	if serverHost == core.LOCALHOST {
		containers, err = cr.SearchContainerByFilter(map[string]string{
			"label": "beast.instance=true",
		})
	} else {
		server := config.Cfg.AvailableServers[serverHost]
		containers, err = remoteManager.SearchContainerByFilterRemote(map[string]string{
			"label": "beast.instance=true",
		}, server)
	}

	if err != nil {
		log.Warnf("Failed to search for instance containers on %s: %v", serverHost, err)
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

		_, err := cache.GetInstance(instanceID)
		if err != nil {
			containerName := ""
			if len(container.Names) > 0 {
				containerName = strings.TrimPrefix(container.Names[0], "/")
			}
			log.Infof("Removing orphaned instance container: %s (instance %s) on %s", containerName, instanceID, serverHost)

			if serverHost == core.LOCALHOST {
				if err := cr.StopAndRemoveContainer(container.ID); err != nil {
					log.Warnf("Failed to remove orphaned container %s: %v", container.ID[:12], err)
				}
			} else {
				server := config.Cfg.AvailableServers[serverHost]
				if err := remoteManager.StopAndRemoveContainerRemote(container.ID, server); err != nil {
					log.Warnf("Failed to remove orphaned container %s on %s: %v", container.ID[:12], serverHost, err)
				}
			}

			cache.FreeContainerPortsOnHost(serverHost, container.ID)
		}
	}
}

// cleanupOrphanedComposeInstancesOnServer finds and removes orphaned docker compose instance projects.
// Docker Compose containers don't have the beast.instance labels, but they have
// com.docker.compose.project labels with project names starting with "beast-instance-".
func cleanupOrphanedComposeInstancesOnServer(serverHost string) {
	var projectNames []string
	var err error

	if serverHost == core.LOCALHOST {
		projectNames, err = getOrphanedComposeInstanceProjects()
	} else {
		server := config.Cfg.AvailableServers[serverHost]
		projectNames, err = getOrphanedComposeInstanceProjectsRemote(server)
	}

	if err != nil {
		log.Warnf("Failed to get compose instance projects on %s: %v", serverHost, err)
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
			log.Infof("Removing orphaned compose instance project: %s (instance %s) on %s", projectName, instanceID, serverHost)

			if serverHost == core.LOCALHOST {
				if err := composeDownProject(projectName); err != nil {
					log.Warnf("Failed to remove orphaned compose project %s: %v", projectName, err)
				}
			} else {
				server := config.Cfg.AvailableServers[serverHost]
				if err := composeDownProjectRemote(projectName, server); err != nil {
					log.Warnf("Failed to remove orphaned compose project %s on %s: %v", projectName, serverHost, err)
				}
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

// getOrphanedComposeInstanceProjectsRemote returns compose instance projects on a remote server
func getOrphanedComposeInstanceProjectsRemote(server config.AvailableServer) ([]string, error) {
	output, err := remoteManager.RunCommandOnServer(server, "docker compose ls --format json")
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

// composeDownProjectRemote removes a docker compose project on a remote server
func composeDownProjectRemote(projectName string, server config.AvailableServer) error {
	cmd := fmt.Sprintf("docker compose -p %s down --remove-orphans -v", projectName)
	output, err := remoteManager.RunCommandOnServer(server, cmd)
	if err != nil {
		return fmt.Errorf("docker compose down failed on remote: %v, output: %s", err, output)
	}

	log.Debugf("Successfully removed compose project %s on %s", projectName, server.Host)
	return nil
}
