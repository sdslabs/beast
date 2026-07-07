package remoteManager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	_ "github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func CreateContainerFromImageRemote(containerConfig cr.CreateContainerConfig, server config.AvailableServer) (string, error) {
	var containerName, containerEnv, exposedPorts, portMap, cpuShareLimit, cpuLimit, memoryLimit, pidLimit, imageID, mountBindings string
	if containerConfig.ContainerName != "" {
		containerName = fmt.Sprintf("--name %s ", containerConfig.ContainerName)
	}
	for _, envVar := range containerConfig.ContainerEnv {
		containerEnv += fmt.Sprintf("--env %s ", envVar)
	}
	for _, portMapping := range containerConfig.PortMapping {
		portMap += fmt.Sprintf("-p 0.0.0.0:%d:%d/%s ", portMapping.HostPort, portMapping.ContainerPort, containerConfig.TrafficType())
		exposedPorts += fmt.Sprintf("--expose %d ", portMapping.ContainerPort)
	}
	if containerConfig.CPUShares != 0 {
		cpuShareLimit = fmt.Sprintf("--cpu-shares %d ", containerConfig.CPUShares)
	}
	if containerConfig.CPUsLimit != 0 {
		cpuLimit = fmt.Sprintf("--cpus %f ", containerConfig.CPUsLimit)
	}
	if containerConfig.Memory != 0 {
		memoryLimit = fmt.Sprintf("--memory %d ", containerConfig.Memory)
	}
	if containerConfig.PidsLimit != 0 {
		pidLimit = fmt.Sprintf("--pids-limit %d ", containerConfig.PidsLimit)
	}
	if containerConfig.ImageId != "" {
		imageID = containerConfig.ImageId
	}
	for src, dest := range containerConfig.MountsMap {
		mountBindings += fmt.Sprintf("--mount type=bind,source=%s,target=%s ", src, dest)
	}
	dockerCommand := fmt.Sprintf("docker run -d %s %s %s %s %s %s %s %s %s %s", containerName, containerEnv, exposedPorts, mountBindings, cpuShareLimit, cpuLimit, memoryLimit, pidLimit, portMap, imageID)
	// fmt.Printf("%s, %s, %s, %s\n", containerName, containerEnv, exposedPorts, portMap)
	// dockerCommand := fmt.Sprintf("docker run \\
	// 	--name <container_name> \\
	// 	--env KEY1=value1 --env KEY2=value2 \\
	// 	--expose <internal_port> \\
	// 	--mount type=bind,source=<host_path>,target=<container_path> \\
	// 	--cpus=<cpu_limit> \\
	// 	--memory=<memory_limit> \\
	// 	--pids-limit <pid_limit> \\
	// 	-p 0.0.0.0:<external_port>:<internal_port> \\
	// 	<image_id>"
	// );
	output, err := RunCommandOnServer(server, dockerCommand)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %s\nOutput: %s", err, output)
	}
	log.Println(output[:12])
	return strings.TrimSpace(output[:12]), nil
}

// Stops and remove cremote container.
// Takes containerID and server config if both is available
// else just take containerID and find the server config from db
func StopAndRemoveContainerRemote(containerId string, server config.AvailableServer) error {
	if server == (config.AvailableServer{}) {
		chall, err := database.QueryChallengeEntries("id", containerId)
		if err != nil {
			if err == (gorm.ErrRecordNotFound) {
				log.Debugf("no container with container id %s present", containerId)
				return nil
			}
			return fmt.Errorf("DATABASE ERROR while fetching challenge details")
		}
		if len(chall) > 0 {
			server = config.Cfg.AvailableServers[chall[0].ServerDeployed]
		} else {
			return fmt.Errorf("no container with container id %s found", containerId)
		}
	}
	stopCommand := fmt.Sprintf("docker stop %s", containerId)
	if _, err := RunCommandOnServer(server, stopCommand); err != nil {
		return fmt.Errorf("failed to stop container on server %s : %w", server.Host, err)
	}
	log.Debugf("Stopped container with ID %s on %s", containerId, server.Host)

	removeCommand := fmt.Sprintf("docker rm --force %s", containerId)
	if _, err := RunCommandOnServer(server, removeCommand); err != nil {
		return fmt.Errorf("failed to remove container on server %s : %w", server.Host, err)
	}
	log.Printf("Removed container with ID %s on %s", containerId, server.Host)

	return nil
}

// Function searches containers based on the filter map on all remote servers
func SearchContainerByFilterRemote(filterMap map[string]string, server config.AvailableServer) ([]types.Container, error) {
	filterArgs := ""
	containers := []types.Container{}
	var output string
	var err error
	for key, val := range filterMap {
		filterArgs += fmt.Sprintf("--filter='%s=%s' ", key, val)
	}
	for serverDeployed, server := range config.Cfg.AvailableServers {
		if server.Active {
			if !config.Cfg.UseLocalDockerDaemon(serverDeployed) {
				output, err = RunCommandOnServer(server, fmt.Sprintf("docker ps -a %s --format '{{.ID}}'", filterArgs))
				if err != nil {
					return []types.Container{}, err
				}
				for _, line := range bytes.Split([]byte(output), []byte("\n")) {
					if len(line) > 0 {
						containers = append(containers, types.Container{ID: string(line)})
					}
				}
			}
		}
	}

	return containers, nil
}

// Function searches for running containers based on the filter map on all remote server
func SearchRunningContainerByFilterRemote(filterMap map[string]string, server config.AvailableServer) ([]types.Container, error) {
	filterArgs := ""
	containers := []types.Container{}
	var output string
	var err error
	for key, val := range filterMap {
		filterArgs += fmt.Sprintf("--filter='%s=%s' ", key, val)
	}
	for serverDeployed, server := range config.Cfg.AvailableServers {
		if server.Active {
			if !config.Cfg.UseLocalDockerDaemon(serverDeployed) {
				output, err = RunCommandOnServer(server, fmt.Sprintf("docker ps %s --format '{{.ID}}'", filterArgs))
				if err != nil {
					return []types.Container{}, err
				}
				for _, line := range bytes.Split([]byte(output), []byte("\n")) {
					if len(line) > 0 {
						containers = append(containers, types.Container{ID: string(line)})
					}
				}
			}
		}
	}

	return containers, nil
}

// Get Containers stdout, stderr logs
func GetContainerStdLogsRemote(containerID string, server config.AvailableServer) (*cr.Log, error) {
	stdoutCmd := fmt.Sprintf("docker logs --details --stdout %s", containerID)
	stderrCmd := fmt.Sprintf("docker logs --details --stderr %s", containerID)

	stdout, err := RunCommandOnServer(server, stdoutCmd)
	if err != nil {
		return nil, fmt.Errorf("error fetching stdout logs: %w", err)
	}

	stderr, err := RunCommandOnServer(server, stderrCmd)
	if err != nil {
		return nil, fmt.Errorf("error fetching stderr logs: %w", err)
	}

	return &cr.Log{Stdout: stdout, Stderr: stderr}, nil
}

// Get live logs of container
func ShowLiveContainerLogsRemote(containerID string, server config.AvailableServer) error {
	command := fmt.Sprintf("docker logs --details --follow %s", containerID)

	output, err := RunCommandOnServer(server, command)
	if err != nil {
		return fmt.Errorf("error streaming live logs: %w", err)
	}

	fmt.Println(output)
	return nil
}

// Commit container on remote server
func CommitContainerRemote(containerID string, server config.AvailableServer) (string, error) {
	command := fmt.Sprintf("docker commit %s", containerID)

	output, err := RunCommandOnServer(server, command)
	if err != nil {
		return "", fmt.Errorf("error committing container: %w", err)
	}
	imageID := strings.TrimSpace(output)
	return imageID, nil
}

func DeployContainerFromComposeRemote(challengeName string, projectName string, stagedDir string, composeFileName string, server config.AvailableServer, ports map[string]uint32) (string, error) {
	extractDir := filepath.Join(stagedDir, challengeName)
	composeFile := filepath.Join(extractDir, composeFileName)

	upCommand := fmt.Sprintf("%s docker compose -f %s -p %s up -d", utils.PortMappingToEnvironmentVariable(ports), composeFile, projectName)
	log.Debugf("Deploying challenge %s using docker compose remotely with project %s and file %s", challengeName, projectName, composeFileName)
	upOutput, err := RunCommandOnServer(server, upCommand)
	if err != nil {
		log.Errorf("docker compose up failed for challenge %s. Output:\n%s", challengeName, upOutput)
		return "", fmt.Errorf("error while running docker compose up on remote: %v", err)
	}

	if err := validateAllComposeServicesRunningRemote(projectName, challengeName, server); err != nil {
		return "", err
	}

	primaryContainerId, err := getPrimaryComposeContainerIdRemote(projectName, server)
	if err != nil {
		log.Warnf("Could not get primary container ID for challenge %s on remote: %v", challengeName, err)
		return "", nil // Return empty string but success
	}

	log.Debugf("Verified challenge %s services are running on remote. Primary container: %s", challengeName, primaryContainerId)
	return primaryContainerId, nil
}

func validateAllComposeServicesRunningRemote(projectName, challengeName string, server config.AvailableServer) error {
	psCommand := fmt.Sprintf("docker compose -p %s ps --format json", projectName)
	log.Debugf("Verifying docker compose services for challenge %s: %s", challengeName, psCommand)
	psOutput, err := RunCommandOnServer(server, psCommand)
	if err != nil {
		log.Errorf("docker compose ps failed for challenge %s. Output:\n%s", challengeName, psOutput)
		return fmt.Errorf("error while verifying docker compose services on remote: %v", err)
	}

	output := strings.TrimSpace(psOutput)
	if output == "" {
		return fmt.Errorf("no services found after compose up for challenge %s", challengeName)
	}

	type ComposeService struct {
		ID      string `json:"ID"`
		Name    string `json:"Name"`
		Service string `json:"Service"`
		State   string `json:"State"`
		Status  string `json:"Status"`
	}

	var services []ComposeService
	var notRunningServices []string

	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var service ComposeService
		if err := json.Unmarshal([]byte(line), &service); err != nil {
			log.Warnf("Failed to parse compose service JSON on remote: %v. Line: %s", err, line)
			continue
		}

		services = append(services, service)

		if service.State != "running" && !strings.HasPrefix(service.Status, "Up") {
			notRunningServices = append(notRunningServices, service.Service)
			log.Warnf("Remote service %s is not running. State: %s, Status: %s", service.Service, service.State, service.Status)
		} else {
			log.Debugf("Remote service %s (name: %s) is running with status: %s", service.Service, service.Name, service.Status)
		}
	}

	if len(services) == 0 {
		return fmt.Errorf("no services detected for challenge %s on remote", challengeName)
	}

	if len(notRunningServices) > 0 {
		return fmt.Errorf("services not running for challenge %s on remote: %v", challengeName, notRunningServices)
	}

	log.Debugf("Verified all %d services are running for challenge %s on remote", len(services), challengeName)
	return nil
}

// gets the first container ID from a compose project on remote
func getPrimaryComposeContainerIdRemote(projectName string, server config.AvailableServer) (string, error) {
	psCommand := fmt.Sprintf("docker compose -p %s ps -q | head -1", projectName)
	output, err := RunCommandOnServer(server, psCommand)
	if err != nil {
		return "", fmt.Errorf("failed to get container IDs on remote: %v", err)
	}

	containerId := strings.TrimSpace(output)
	if containerId == "" {
		return "", fmt.Errorf("no containers found for project %s on remote", projectName)
	}

	// Return first 12 characters
	if len(containerId) >= 12 {
		return containerId[:12], nil
	}
	return containerId, nil
}

// ComposeDownProjectRemote runs docker compose down for an explicit -p project name.
func ComposeDownProjectRemote(projectName string, server config.AvailableServer) error {
	log.Debugf("Stopping docker compose project %s on remote", projectName)
	downCommand := fmt.Sprintf("docker compose -p %s down", projectName)
	downOutput, err := RunCommandOnServer(server, downCommand)
	if err != nil {
		return fmt.Errorf("docker compose down failed for project %s on remote: %v. Output: %s", projectName, err, downOutput)
	}
	log.Debugf("Successfully stopped compose project %s on remote. Output: %s", projectName, downOutput)
	return nil
}

// ComposePurgeProjectRemote purges a compose project by explicit -p name (shared or instanced).
func ComposePurgeProjectRemote(projectName string, server config.AvailableServer) error {
	log.Debugf("Purging docker compose project %s on remote", projectName)
	purgeCommand := fmt.Sprintf("docker compose -p %s down --remove-orphans --volumes --rmi all", projectName)
	purgeOutput, err := RunCommandOnServer(server, purgeCommand)
	if err != nil {
		return fmt.Errorf("docker compose purge failed for project %s on remote: %v. Output: %s", projectName, err, purgeOutput)
	}
	log.Debugf("Successfully purged compose project %s on remote. Output: %s", projectName, purgeOutput)
	return nil
}
