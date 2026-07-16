package remoteManager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	_ "github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func CreateContainerFromImageRemote(containerConfig cr.CreateContainerConfig, server config.AvailableServer) (string, error) {
	arguments := []string{"docker", "run", "-d"}
	if containerConfig.ContainerName != "" {
		arguments = append(arguments, "--name", containerConfig.ContainerName)
	}
	for _, envVar := range containerConfig.ContainerEnv {
		arguments = append(arguments, "--env", envVar)
	}
	for _, portMapping := range containerConfig.PortMapping {
		arguments = append(arguments,
			"--publish", fmt.Sprintf("0.0.0.0:%d:%d/%s", portMapping.HostPort, portMapping.ContainerPort, containerConfig.TrafficType()),
			"--expose", strconv.FormatUint(uint64(portMapping.ContainerPort), 10))
	}
	if containerConfig.CPUShares != 0 {
		arguments = append(arguments, "--cpu-shares", strconv.FormatInt(containerConfig.CPUShares, 10))
	}
	if containerConfig.CPUsLimit != 0 {
		arguments = append(arguments, "--cpus", strconv.FormatFloat(float64(containerConfig.CPUsLimit), 'f', -1, 32))
	}
	if containerConfig.Memory != 0 {
		arguments = append(arguments, "--memory", strconv.FormatInt(containerConfig.Memory, 10))
	}
	if containerConfig.PidsLimit != 0 {
		arguments = append(arguments, "--pids-limit", strconv.FormatInt(containerConfig.PidsLimit, 10))
	}
	if containerConfig.ImageId == "" {
		return "", fmt.Errorf("image ID is required")
	}
	mountSources := make([]string, 0, len(containerConfig.MountsMap))
	for source := range containerConfig.MountsMap {
		mountSources = append(mountSources, source)
	}
	sort.Strings(mountSources)
	for _, source := range mountSources {
		arguments = append(arguments, "--mount", fmt.Sprintf("type=bind,source=%s,target=%s", source, containerConfig.MountsMap[source]))
	}
	arguments = append(arguments, containerConfig.ImageId)
	output, err := RunArgsOnServer(server, arguments...)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %s\nOutput: %s", err, output)
	}
	containerID := strings.TrimSpace(output)
	if len(containerID) < 12 {
		return "", fmt.Errorf("docker returned invalid container ID %q", containerID)
	}
	log.Println(containerID[:12])
	return containerID[:12], nil
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
	if _, err := RunArgsOnServer(server, "docker", "stop", containerId); err != nil {
		return fmt.Errorf("failed to stop container on server %s : %w", server.Host, err)
	}
	log.Debugf("Stopped container with ID %s on %s", containerId, server.Host)

	if _, err := RunArgsOnServer(server, "docker", "rm", "--force", containerId); err != nil {
		return fmt.Errorf("failed to remove container on server %s : %w", server.Host, err)
	}
	log.Printf("Removed container with ID %s on %s", containerId, server.Host)

	return nil
}

// Function searches containers based on the filter map on all remote servers
func SearchContainerByFilterRemote(filterMap map[string]string, server config.AvailableServer) ([]types.Container, error) {
	return searchContainersRemote(filterMap, server, true)
}

// Function searches for running containers based on the filter map on all remote server
func SearchRunningContainerByFilterRemote(filterMap map[string]string, server config.AvailableServer) ([]types.Container, error) {
	return searchContainersRemote(filterMap, server, false)
}

func searchContainersRemote(filterMap map[string]string, server config.AvailableServer, all bool) ([]types.Container, error) {
	arguments := []string{"docker", "ps"}
	if all {
		arguments = append(arguments, "--all")
	}
	keys := make([]string, 0, len(filterMap))
	for key := range filterMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		arguments = append(arguments, "--filter", key+"="+filterMap[key])
	}
	arguments = append(arguments, "--format", "{{json .}}")
	output, err := RunArgsOnServer(server, arguments...)
	if err != nil {
		return nil, err
	}
	type containerRow struct {
		ID     string
		Names  string
		Labels string
	}
	containers := make([]types.Container, 0)
	for _, line := range bytes.Split([]byte(output), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var row containerRow
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, fmt.Errorf("parse remote container list: %w", err)
		}
		labels := make(map[string]string)
		for _, label := range strings.Split(row.Labels, ",") {
			key, value, found := strings.Cut(label, "=")
			if found {
				labels[key] = value
			}
		}
		containers = append(containers, types.Container{ID: row.ID, Names: []string{row.Names}, Labels: labels})
	}
	return containers, nil
}

// Get Containers stdout, stderr logs
func GetContainerStdLogsRemote(containerID string, server config.AvailableServer) (*cr.Log, error) {
	stdout, err := RunArgsOnServer(server, "docker", "logs", "--details", containerID)
	if err != nil {
		return nil, fmt.Errorf("error fetching stdout logs: %w", err)
	}

	return &cr.Log{Stdout: stdout}, nil
}

// Get live logs of container
func ShowLiveContainerLogsRemote(containerID string, server config.AvailableServer) error {
	output, err := RunArgsOnServer(server, "docker", "logs", "--details", "--follow", containerID)
	if err != nil {
		return fmt.Errorf("error streaming live logs: %w", err)
	}

	fmt.Println(output)
	return nil
}

// Commit container on remote server
func CommitContainerRemote(containerID string, server config.AvailableServer) (string, error) {
	output, err := RunArgsOnServer(server, "docker", "commit", containerID)
	if err != nil {
		return "", fmt.Errorf("error committing container: %w", err)
	}
	imageID := strings.TrimSpace(output)
	return imageID, nil
}

func DeployContainerFromComposeRemote(challengeName string, projectName string, stagedDir string, composeFileName string, server config.AvailableServer, ports map[string]uint32) (string, error) {
	extractDir := filepath.Join(stagedDir, challengeName)
	composeFile := filepath.Join(extractDir, composeFileName)

	environment := make(map[string]string, len(ports))
	for variable, port := range ports {
		environment[variable] = strconv.FormatUint(uint64(port), 10)
	}
	log.Debugf("Deploying challenge %s using docker compose remotely with project %s and file %s", challengeName, projectName, composeFileName)
	upOutput, err := RunArgsWithEnvOnServer(server, environment, "docker", "compose", "-f", composeFile, "-p", projectName, "up", "-d")
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
	log.Debugf("Verifying docker compose services for challenge %s", challengeName)
	psOutput, err := RunArgsOnServer(server, "docker", "compose", "-p", projectName, "ps", "--format", "json")
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
	output, err := RunArgsOnServer(server, "docker", "compose", "-p", projectName, "ps", "-q")
	if err != nil {
		return "", fmt.Errorf("failed to get container IDs on remote: %v", err)
	}

	containerIDs := strings.Fields(output)
	if len(containerIDs) == 0 {
		return "", fmt.Errorf("no containers found for project %s on remote", projectName)
	}
	containerId := containerIDs[0]

	// Return first 12 characters
	if len(containerId) >= 12 {
		return containerId[:12], nil
	}
	return containerId, nil
}

// ComposeDownProjectRemote runs docker compose down for an explicit -p project name.
func ComposeDownProjectRemote(projectName string, server config.AvailableServer) error {
	log.Debugf("Stopping docker compose project %s on remote", projectName)
	downOutput, err := RunArgsOnServer(server, "docker", "compose", "-p", projectName, "down")
	if err != nil {
		return fmt.Errorf("docker compose down failed for project %s on remote: %v. Output: %s", projectName, err, downOutput)
	}
	log.Debugf("Successfully stopped compose project %s on remote. Output: %s", projectName, downOutput)
	return nil
}

// ComposePurgeProjectRemote purges a compose project by explicit -p name (shared or instanced).
func ComposePurgeProjectRemote(projectName string, server config.AvailableServer) error {
	log.Debugf("Purging docker compose project %s on remote", projectName)
	purgeOutput, err := RunArgsOnServer(server, "docker", "compose", "-p", projectName, "down", "--remove-orphans", "--volumes", "--rmi", "all")
	if err != nil {
		return fmt.Errorf("docker compose purge failed for project %s on remote: %v. Output: %s", projectName, err, purgeOutput)
	}
	log.Debugf("Successfully purged compose project %s on remote. Output: %s", projectName, purgeOutput)
	return nil
}
