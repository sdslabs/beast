package remoteManager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func CreateContainerFromImageRemote(containerConfig cr.CreateContainerConfig, server config.AvailableServer) (string, error) {
	var containerName, containerEnv, exposedPorts, portMap, cpuShareLimit, cpuLimit, memoryLimit, pidLimit, imageID, mountBindings, containerLabels, networkArg string
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
	if containerConfig.ContainerNetwork != "" {
		networkArg = fmt.Sprintf("--network %s ", containerConfig.ContainerNetwork)
	}
	labels := map[string]string{
		"beast.challenge":             containerConfig.ChallengeName,
		"com.sdslabs.beast.project":   utils.ProjectNameNotInstanced(containerConfig.ChallengeName),
		"com.docker.compose.project":  utils.ProjectNameNotInstanced(containerConfig.ChallengeName),
		"com.sdslabs.beast.challenge": containerConfig.ChallengeName,
	}
	for k, v := range containerConfig.Labels {
		labels[k] = v
	}
	for k, v := range labels {
		containerLabels += fmt.Sprintf("--label %s=%s ", k, v)
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
	dockerCommand := fmt.Sprintf("docker run -d %s %s %s %s %s %s %s %s %s %s %s %s", containerName, networkArg, containerLabels, containerEnv, exposedPorts, mountBindings, cpuShareLimit, cpuLimit, memoryLimit, pidLimit, portMap, imageID)
	output, err := RunCommandOnServer(server, dockerCommand)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %s\nOutput: %s", err, output)
	}
	trimmed := strings.TrimSpace(output)
	if len(trimmed) > 12 {
		trimmed = trimmed[:12]
	}
	log.Println(trimmed)
	return trimmed, nil
}

// Stops and removes remote container.
// Takes containerID and server config if both is available
// else just take containerID and find the server config from db
func StopAndRemoveContainerRemote(containerId string, server config.AvailableServer) error {
	if server == (config.AvailableServer{}) {
		chall, err := database.QueryChallengeEntries("container_id", containerId)
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

func parseRemoteContainerLines(output string) []types.Container {
	var containers []types.Container
	for _, raw := range strings.Split(strings.TrimSpace(output), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		c := types.Container{ID: strings.TrimSpace(parts[0])}
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
			c.Names = strings.Split(parts[1], ",")
		}
		if len(parts) == 3 && strings.TrimSpace(parts[2]) != "" {
			labels := make(map[string]string)
			for _, kv := range strings.Split(parts[2], ",") {
				if pair := strings.SplitN(kv, "=", 2); len(pair) == 2 {
					labels[strings.TrimSpace(pair[0])] = strings.TrimSpace(pair[1])
				}
			}
			c.Labels = labels
		}
		containers = append(containers, c)
	}
	return containers
}

func searchContainersRemoteWithFlag(filterMap map[string]string, server config.AvailableServer, allFlag string) ([]types.Container, error) {
	filterArgs := ""
	for key, val := range filterMap {
		filterArgs += fmt.Sprintf("--filter='%s=%s' ", key, val)
	}
	psCmd := fmt.Sprintf("docker ps %s %s --format '{{.ID}}\t{{.Names}}\t{{.Labels}}'", allFlag, filterArgs)
	if server != (config.AvailableServer{}) {
		output, err := RunCommandOnServer(server, psCmd)
		if err != nil {
			return []types.Container{}, err
		}
		return parseRemoteContainerLines(output), nil
	}

	var containers []types.Container
	for serverDeployed, s := range config.Cfg.AvailableServers {
		if s.Active && !config.Cfg.UseLocalDockerDaemon(serverDeployed) {
			output, err := RunCommandOnServer(s, psCmd)
			if err != nil {
				return []types.Container{}, err
			}
			containers = append(containers, parseRemoteContainerLines(output)...)
		}
	}
	return containers, nil
}

// SearchContainerByFilterRemote searches all containers matching filterMap on the specified remote server
// (or all active remote servers if server is zero-value).
func SearchContainerByFilterRemote(filterMap map[string]string, server config.AvailableServer) ([]types.Container, error) {
	return searchContainersRemoteWithFlag(filterMap, server, "-a")
}

// SearchRunningContainerByFilterRemote searches running containers matching filterMap on the specified remote server
// (or all active remote servers if server is zero-value).
func SearchRunningContainerByFilterRemote(filterMap map[string]string, server config.AvailableServer) ([]types.Container, error) {
	return searchContainersRemoteWithFlag(filterMap, server, "")
}

// VerifyComposeProjectRunningRemote verifies all services in a remote compose project are running.
func VerifyComposeProjectRunningRemote(projectName string, server config.AvailableServer) error {
	return validateAllComposeServicesRunningRemote(projectName, projectName, server)
}

// GetContainerStdLogsRemote returns the separate stdout and stderr streams.
func GetContainerStdLogsRemote(containerID string, server config.AvailableServer) (*cr.Log, error) {
	var stdout, stderr bytes.Buffer
	if err := writeContainerLogsRemote(containerID, server, false, &stdout, &stderr); err != nil {
		return nil, err
	}
	return &cr.Log{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func ShowLiveContainerLogsRemote(containerID string, server config.AvailableServer) error {
	return writeContainerLogsRemote(containerID, server, true, os.Stdout, os.Stderr)
}

func writeContainerLogsRemote(containerID string, server config.AvailableServer, follow bool, stdout, stderr io.Writer) error {
	if !server.Active {
		return fmt.Errorf("server is inactive in config.toml")
	}
	if !utils.IsShortID(containerID) && utils.ValidateID(containerID) != nil {
		return fmt.Errorf("invalid container ID")
	}
	client, err := CreateSSHClient(server)
	if err != nil {
		return err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdout = stdout
	session.Stderr = stderr

	command := "docker logs --details"
	if follow {
		command += " --follow"
	}
	return session.Run(command + " " + containerID)
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
		return "", fmt.Errorf("get primary container for remote challenge %s: %w", challengeName, err)
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

func getPrimaryComposeContainerIdRemote(projectName string, server config.AvailableServer) (string, error) {
	psCommand := fmt.Sprintf("docker compose -p %s ps -q", projectName)
	if _, instanced := utils.ExtractInstanceIDFromComposeProjectName(projectName); instanced {
		psCommand += " " + core.SSH_CONTAINER_COMPOSE
	}
	output, err := RunCommandOnServer(server, psCommand)
	if err != nil {
		return "", fmt.Errorf("list containers for remote Compose project %s: %w", projectName, err)
	}

	containerIds := strings.Fields(strings.TrimSpace(output))
	if len(containerIds) == 0 {
		return "", fmt.Errorf("no primary container found in remote Compose project %s", projectName)
	}

	containerId := containerIds[0]
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
