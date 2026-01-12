package cr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/sdslabs/beastv4/pkg/defaults"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type PortMapping struct {
	HostPort      uint32
	ContainerPort uint32
}

// TrafficType is the protocol supported by container ingress and egress through
// the port mappings.
type TrafficType string

// String returns the string representation of the traffic type.
func (t TrafficType) String() string {
	return string(t)
}

const (
	TCPTraffic TrafficType = "tcp"
	UDPTraffic TrafficType = "udp"

	DefaultTraffic TrafficType = TCPTraffic
)

func IsValidTrafficType(t string) bool {
	switch TrafficType(t) {
	case TCPTraffic, UDPTraffic:
		return true
	default:
		return false
	}
}

func GetValidTrafficTypes() []string {
	return []string{UDPTraffic.String(), TCPTraffic.String()}
}

type CreateContainerConfig struct {
	PortMapping      []PortMapping
	MountsMap        map[string]string
	ImageId          string
	ContainerName    string
	ChallengeName    string
	ContainerEnv     []string
	ContainerNetwork string
	Traffic          TrafficType

	CPUShares int64
	Memory    int64
	PidsLimit int64
}

func (c *CreateContainerConfig) TrafficType() string {
	if c.Traffic.String() == "" {
		return DefaultTraffic.String()
	}

	return c.Traffic.String()
}

type Log struct {
	Stderr string
	Stdout string
}

// Function is equivalent to docker ps -a
func SearchContainerByFilter(filterMap map[string]string) ([]types.Container, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return []types.Container{}, err
	}

	filterArgs := filters.NewArgs()
	for key, val := range filterMap {
		filterArgs.Add(key, val)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})

	return containers, err
}

// Function is equivalent to docker ps
func SearchRunningContainerByFilter(filterMap map[string]string) ([]types.Container, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return []types.Container{}, err
	}

	filterArgs := filters.NewArgs()
	for key, val := range filterMap {
		filterArgs.Add(key, val)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filterArgs,
	})

	return containers, err
}

func StopAndRemoveContainer(containerId string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	// Try to stop using default timeout we are using for beast
	err = cli.ContainerStop(context.Background(), containerId, &defaults.DefaultDockerStopTimeout)
	if err != nil {
		return err
	}
	log.Debug("Stopped container with ID ", containerId)

	log.Debug("Removing container with ID ", containerId)
	err = cli.ContainerRemove(context.Background(), containerId, types.ContainerRemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   false,
		Force:         true,
	})

	return err
}

func CreateContainerFromImage(containerConfig *CreateContainerConfig) (string, error) {
	containerName := fmt.Sprintf("beast_%s_%s", containerConfig.ChallengeName, containerConfig.ContainerName[:3])
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}

	portSet := make(nat.PortSet)
	portMap := make(nat.PortMap)

	for _, portMapping := range containerConfig.PortMapping {
		natPort, err := nat.NewPort(containerConfig.TrafficType(), strconv.Itoa(int(portMapping.ContainerPort)))
		if err != nil {
			return "", fmt.Errorf("error while creating new port from port %d", portMapping.ContainerPort)
		}

		portSet[natPort] = struct{}{}

		portMap[natPort] = []nat.PortBinding{{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(int(portMapping.HostPort)),
		}}
	}

	config := &container.Config{
		Image:        containerConfig.ImageId,
		ExposedPorts: portSet,
		Env:          containerConfig.ContainerEnv,
	}

	var mountBindings []mount.Mount
	for src, dest := range containerConfig.MountsMap {
		mnt := mount.Mount{
			Type:   mount.TypeBind,
			Source: src,
			Target: dest,
		}

		mountBindings = append(mountBindings, mnt)
	}

	resources := container.Resources{
		CPUShares: containerConfig.CPUShares,
		Memory:    containerConfig.Memory,
		PidsLimit: &containerConfig.PidsLimit,
	}

	hostConfig := &container.HostConfig{
		PortBindings: portMap,
		Mounts:       mountBindings,
		NetworkMode:  container.NetworkMode(containerConfig.ContainerNetwork),
		Resources:    resources,
	}

	createResp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		log.Errorf("Error while creating the container with name %s", containerName)
		return "", err
	}

	containerId := createResp.ID
	if len(createResp.Warnings) > 0 {
		log.Warnf("Warnings while creating the container : %s", createResp.Warnings)
	}

	if err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		log.Errorf("Error while starting the container : %s", err)
		return "", err
	}

	return containerId, nil
}

func GetContainerStdLogs(containerID string) (*Log, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	stdout, err := cli.ContainerLogs(context.Background(), containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		Details:    true,
	})
	if err != nil {
		return nil, err
	}
	defer stdout.Close()

	stdoutlogs, _ := ioutil.ReadAll(stdout)

	stderr, err := cli.ContainerLogs(context.Background(), containerID, types.ContainerLogsOptions{
		ShowStderr: true,
		Details:    true,
	})
	if err != nil {
		return nil, err
	}
	defer stderr.Close()

	stderrlogs, _ := ioutil.ReadAll(stderr)

	return &Log{Stdout: string(stdoutlogs), Stderr: string(stderrlogs)}, nil
}

func ShowLiveContainerLogs(containerID string) {
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error(err)
	}

	stream, err := cli.ContainerLogs(context.Background(), containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Details:    true,
	})
	if err != nil {
		log.Error(err)
	}
	defer stream.Close()

	logs, _ := ioutil.ReadAll(stream)
	fmt.Println(string(logs))
}

func CommitContainer(containerId string) (string, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}

	commitResp, err := cli.ContainerCommit(ctx, containerId, types.ContainerCommitOptions{})
	if err != nil {
		return "", err
	}

	return commitResp.ID, nil
}

func DeployContainerFromCompose(challengeName, stagedPath, composeFileName string) (string, error) {
	extractDir := filepath.Join(stagedPath, challengeName)
	projectName := fmt.Sprintf("beast-%s", challengeName)
	composeFile := filepath.Join(extractDir, composeFileName)

	log.Debugf("Deploying challenge %s using docker compose with project name %s and file %s", challengeName, projectName, composeFileName)

	// Deploy with project name - Docker Compose automatically labels containers with
	// com.docker.compose.project=<projectName>
	upCmd := exec.Command("docker", "compose",
		"-f", composeFile,
		"-p", projectName,
		"up", "-d")

	var upOutput bytes.Buffer
	upCmd.Stdout = &upOutput
	upCmd.Stderr = &upOutput

	if err := upCmd.Run(); err != nil {
		log.Errorf("docker compose up failed for challenge %s. Output:\n%s", challengeName, upOutput.String())
		return "", fmt.Errorf("error while running docker compose up: %v", err)
	}

	if err := validateAllComposeServicesRunning(projectName, challengeName); err != nil {
		return "", err
	}

	primaryContainerId, err := getPrimaryComposeContainerId(projectName)
	if err != nil {
		log.Warnf("Could not get primary container ID for challenge %s: %v", challengeName, err)
		return "", nil // Return empty string but success
	}

	log.Debugf("Verified challenge %s services are running. Primary container: %s", challengeName, primaryContainerId)
	return primaryContainerId, nil
}

func validateAllComposeServicesRunning(projectName, challengeName string) error {
	psCmd := exec.Command("docker", "compose", "-p", projectName, "ps", "--format", "json")
	var psOutput bytes.Buffer
	psCmd.Stdout = &psOutput
	psCmd.Stderr = &psOutput

	if err := psCmd.Run(); err != nil {
		return fmt.Errorf("error checking container status after compose up for challenge %s. Output:\n%s", challengeName, psOutput.String())
	}

	output := strings.TrimSpace(psOutput.String())
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
			log.Warnf("Failed to parse compose service JSON: %v. Line: %s", err, line)
			continue
		}

		services = append(services, service)

		if service.State != "running" && !strings.HasPrefix(service.Status, "Up") {
			notRunningServices = append(notRunningServices, service.Service)
			log.Warnf("Service %s is not running. State: %s, Status: %s", service.Service, service.State, service.Status)
		} else {
			log.Debugf("Service %s (name: %s) is running with status: %s", service.Service, service.Name, service.Status)
		}
	}

	if len(services) == 0 {
		return fmt.Errorf("no services detected for challenge %s", challengeName)
	}

	if len(notRunningServices) > 0 {
		return fmt.Errorf("services not running for challenge %s: %v", challengeName, notRunningServices)
	}

	log.Debugf("Verified all %d services are running for challenge %s", len(services), challengeName)
	return nil
}

// gets the first container ID from a compose project
func getPrimaryComposeContainerId(projectName string) (string, error) {
	psCmd := exec.Command("docker", "compose", "-p", projectName, "ps", "-q")
	var output bytes.Buffer
	psCmd.Stdout = &output

	if err := psCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get container IDs: %v", err)
	}

	containerIds := strings.Fields(strings.TrimSpace(output.String()))
	if len(containerIds) == 0 {
		return "", fmt.Errorf("no containers found for project %s", projectName)
	}

	// Return first 12 characters of the first container ID
	if len(containerIds[0]) >= 12 {
		return containerIds[0][:12], nil
	}
	return containerIds[0], nil
}

func ComposeDown(challengeName, stagedDir string) error {
	log.Debugf("Stopping challenge %s using docker compose", challengeName)
	projectName := fmt.Sprintf("beast-%s", challengeName)

	// Try using project name
	downCmd := exec.Command("docker", "compose", "-p", projectName, "down")
	var downOutput bytes.Buffer
	downCmd.Stdout = &downOutput
	downCmd.Stderr = &downOutput

	if err := downCmd.Run(); err != nil {
		log.Warnf("docker compose down with project name failed for challenge %s: %v. Trying label-based cleanup...", challengeName, err)
		// Fallback to label-based cleanup
		return cleanupComposeByLabels(challengeName)
	}

	log.Debugf("Successfully stopped challenge %s", challengeName)
	return nil
}

func cleanupComposeByLabels(challengeName string) error {
	log.Debugf("Using label-based cleanup for challenge %s", challengeName)

	// Find all containers with beast.challenge label
	findCmd := exec.Command("docker", "ps", "-aq",
		"--filter", fmt.Sprintf("label=beast.challenge=%s", challengeName))

	var output bytes.Buffer
	findCmd.Stdout = &output

	if err := findCmd.Run(); err != nil {
		return fmt.Errorf("error finding containers by label: %v", err)
	}

	containerIds := strings.Fields(strings.TrimSpace(output.String()))
	if len(containerIds) == 0 {
		log.Debugf("No containers found for challenge %s", challengeName)
		return nil
	}

	log.Debugf("Found %d containers to remove for challenge %s", len(containerIds), challengeName)

	removeCmd := exec.Command("docker", "rm", "-f")
	removeCmd.Args = append(removeCmd.Args, containerIds...)

	var removeOutput bytes.Buffer
	removeCmd.Stdout = &removeOutput
	removeCmd.Stderr = &removeOutput

	if err := removeCmd.Run(); err != nil {
		return fmt.Errorf("error removing containers: %v. Output: %s", err, removeOutput.String())
	}

	log.Debugf("Successfully removed containers for challenge %s using label-based cleanup", challengeName)
	return nil
}

func ComposePurge(challengeName, stagedDir string) error {
	log.Debugf("Purging challenge %s using docker compose", challengeName)
	projectName := fmt.Sprintf("beast-%s", challengeName)

	// Try using project name with full cleanup
	purgeCmd := exec.Command("docker", "compose", "-p", projectName,
		"down", "--remove-orphans", "--volumes", "--rmi", "all")

	var purgeOutput bytes.Buffer
	purgeCmd.Stdout = &purgeOutput
	purgeCmd.Stderr = &purgeOutput

	if err := purgeCmd.Run(); err != nil {
		log.Warnf("docker compose purge with project name failed for challenge %s: %v. Trying label-based cleanup...", challengeName, err)
		// Fallback to label-based cleanup
		if err := cleanupComposeByLabels(challengeName); err != nil {
			return err
		}
		cleanupComposeVolumesAndNetworks(projectName)
	}

	log.Debugf("Successfully purged challenge %s", challengeName)
	return nil
}

func cleanupComposeVolumesAndNetworks(projectName string) {
	volCmd := exec.Command("docker", "volume", "ls", "-q",
		"--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))

	var volOutput bytes.Buffer
	volCmd.Stdout = &volOutput

	if err := volCmd.Run(); err == nil {
		volumes := strings.Fields(strings.TrimSpace(volOutput.String()))
		if len(volumes) > 0 {
			removeVolCmd := exec.Command("docker", "volume", "rm")
			removeVolCmd.Args = append(removeVolCmd.Args, volumes...)
			if err := removeVolCmd.Run(); err != nil {
				log.Warnf("Failed to remove volumes for project %s: %v", projectName, err)
			}
		}
	}

	netCmd := exec.Command("docker", "network", "ls", "-q",
		"--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))

	var netOutput bytes.Buffer
	netCmd.Stdout = &netOutput

	if err := netCmd.Run(); err == nil {
		networks := strings.Fields(strings.TrimSpace(netOutput.String()))
		if len(networks) > 0 {
			removeNetCmd := exec.Command("docker", "network", "rm")
			removeNetCmd.Args = append(removeNetCmd.Args, networks...)
			if err := removeNetCmd.Run(); err != nil {
				log.Warnf("Failed to remove networks for project %s: %v", projectName, err)
			}
		}
	}
}
