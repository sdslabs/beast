package cr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/sdslabs/beastv4/pkg/defaults"
	utils "github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
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

	maxContainerLogBytes int64 = 4 << 20
)

var errContainerLogLimit = errors.New("container logs exceed 4 MiB limit")

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
	Labels           map[string]string

	CPUShares int64
	CPUsLimit float32
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
	cli, err := newDockerClient()
	if err != nil {
		return []types.Container{}, err
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPIRequestTimeout)
	defer cancel()

	filterArgs := filters.NewArgs()
	for key, val := range filterMap {
		filterArgs.Add(key, val)
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})

	return containers, err
}

// Function is equivalent to docker ps
func SearchRunningContainerByFilter(filterMap map[string]string) ([]types.Container, error) {
	cli, err := newDockerClient()
	if err != nil {
		return []types.Container{}, err
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPIRequestTimeout)
	defer cancel()

	filterArgs := filters.NewArgs()
	for key, val := range filterMap {
		filterArgs.Add(key, val)
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		Filters: filterArgs,
	})

	return containers, err
}

func StopAndRemoveContainer(containerId string) error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPIRequestTimeout)
	defer cancel()

	// Try to stop using default timeout we are using for beast
	err = cli.ContainerStop(ctx, containerId, &defaults.DefaultDockerStopTimeout)
	if err != nil {
		return err
	}
	log.Debug("Stopped container with ID ", containerId)

	log.Debug("Removing container with ID ", containerId)
	err = cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   false,
		Force:         true,
	})

	return err
}

func CreateContainerFromImage(containerConfig *CreateContainerConfig) (string, error) {
	containerName := containerConfig.ContainerName
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPILongTimeout)
	defer cancel()
	cli, err := newDockerClient()
	if err != nil {
		return "", err
	}
	defer cli.Close()

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

	labels := map[string]string{
		"beast.challenge":             containerConfig.ChallengeName,
		"com.sdslabs.beast.project":   utils.ProjectNameNotInstanced(containerConfig.ChallengeName),
		"com.docker.compose.project":  utils.ProjectNameNotInstanced(containerConfig.ChallengeName),
		"com.sdslabs.beast.challenge": containerConfig.ChallengeName,
	}
	for k, v := range containerConfig.Labels {
		labels[k] = v
	}

	config := &container.Config{
		Image:        containerConfig.ImageId,
		ExposedPorts: portSet,
		Env:          containerConfig.ContainerEnv,
		Labels:       labels,
	}

	mountBindings := readOnlyBindMounts(containerConfig.MountsMap)

	resources := container.Resources{
		NanoCPUs:  int64(containerConfig.CPUsLimit * 1e9),
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
	applyDefaultContainerSecurity(hostConfig)

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
		removeErr := cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{Force: true})
		if removeErr != nil {
			log.Errorf("Error while removing failed container %s: %s", containerId, removeErr)
		}
		log.Errorf("Error while starting the container : %s", err)
		return "", err
	}

	return containerId, nil
}

func readOnlyBindMounts(mounts map[string]string) []mount.Mount {
	bindings := make([]mount.Mount, 0, len(mounts))
	for src, dest := range mounts {
		bindings = append(bindings, mount.Mount{
			Type:     mount.TypeBind,
			Source:   src,
			Target:   dest,
			ReadOnly: true,
		})
	}
	return bindings
}

func applyDefaultContainerSecurity(hostConfig *container.HostConfig) {
	hostConfig.CapDrop = []string{"ALL"}
	hostConfig.SecurityOpt = []string{"no-new-privileges"}
}

func GetContainerStdLogs(containerID string) (*Log, error) {
	cli, err := newDockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPIRequestTimeout)
	defer cancel()

	stdout, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		Details:    true,
	})
	if err != nil {
		return nil, err
	}
	defer stdout.Close()

	stdoutlogs, err := readContainerLogs(stdout, maxContainerLogBytes)
	if err != nil {
		return nil, fmt.Errorf("read container stdout: %w", err)
	}

	stderr, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStderr: true,
		Details:    true,
	})
	if err != nil {
		return nil, err
	}
	defer stderr.Close()

	stderrlogs, err := readContainerLogs(stderr, maxContainerLogBytes-int64(len(stdoutlogs)))
	if err != nil {
		return nil, fmt.Errorf("read container stderr: %w", err)
	}

	return &Log{Stdout: string(stdoutlogs), Stderr: string(stderrlogs)}, nil
}

func readContainerLogs(reader io.Reader, limit int64) ([]byte, error) {
	logs, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(logs)) > limit {
		return nil, errContainerLogLimit
	}
	return logs, nil
}

func ShowLiveContainerLogs(containerID string) error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPIRequestTimeout)
	defer cancel()

	stream, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Details:    true,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	logs, err := readContainerLogs(stream, maxContainerLogBytes)
	if err != nil {
		return fmt.Errorf("read container logs: %w", err)
	}
	fmt.Println(string(logs))
	return nil
}

func CommitContainer(containerId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPILongTimeout)
	defer cancel()
	cli, err := newDockerClient()
	if err != nil {
		return "", err
	}
	defer cli.Close()

	commitResp, err := cli.ContainerCommit(ctx, containerId, types.ContainerCommitOptions{})
	if err != nil {
		return "", err
	}

	return commitResp.ID, nil
}

func DeployContainerFromCompose(challengeName string, projectName string, stagedPath string, composeFileName string, ports map[string]uint32) (string, error) {
	extractDir := filepath.Join(stagedPath, challengeName)
	composeFile := filepath.Join(extractDir, composeFileName)

	log.Debugf("Deploying challenge %s using docker compose with project name %s and file %s", challengeName, projectName, composeFileName)

	// Deploy with project name - Docker Compose automatically labels containers with
	// com.docker.compose.project=<projectName>
	arguments := []string{"compose",
		"-f", composeFile,
		"-p", projectName,
		"up", "-d"}

	environment := os.Environ()
	for variable, port := range ports {
		environment = append(environment, fmt.Sprintf("%s=%s", variable, strconv.FormatUint(uint64(port), 10)))
	}

	upOutput, err := runRuntimeCommand("docker", arguments, runtimeCommandOptions{environment: environment})
	if err != nil {
		log.Errorf("docker compose up failed for challenge %s. Output:\n%s", challengeName, upOutput)
		return "", cleanupFailedComposeDeployment(projectName, fmt.Errorf("run docker compose up: %w", err))
	}

	if err := validateAllComposeServicesRunning(projectName, challengeName); err != nil {
		return "", cleanupFailedComposeDeployment(projectName, err)
	}

	primaryContainerId, err := getPrimaryComposeContainerId(projectName)
	if err != nil {
		return "", cleanupFailedComposeDeployment(projectName, fmt.Errorf("get primary container for challenge %s: %w", challengeName, err))
	}

	log.Debugf("Verified challenge %s services are running. Primary container: %s", challengeName, primaryContainerId)
	return primaryContainerId, nil
}

func validateAllComposeServicesRunning(projectName, challengeName string) error {
	psOutput, err := runRuntimeCommand("docker", []string{"compose", "-p", projectName, "ps", "--format", "json"}, runtimeCommandOptions{})
	if err != nil {
		return fmt.Errorf("error checking container status after compose up for challenge %s: %w. Output:\n%s", challengeName, err, psOutput)
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
	output, err := runRuntimeCommand("docker", []string{"compose", "-p", projectName, "ps", "-q"}, runtimeCommandOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get container IDs: %v", err)
	}

	containerIds := strings.Fields(strings.TrimSpace(output))
	if len(containerIds) == 0 {
		return "", fmt.Errorf("no containers found for project %s", projectName)
	}

	// Return first 12 characters of the first container ID
	if len(containerIds[0]) >= 12 {
		return containerIds[0][:12], nil
	}
	return containerIds[0], nil
}

// ComposeDownProject runs docker compose down for an explicit -p project name (shared or instanced).
func ComposeDownProject(projectName string) error {
	log.Debugf("Stopping docker compose project %s", projectName)

	downOutput, err := runRuntimeCommand("docker", []string{"compose", "-p", projectName, "down"}, runtimeCommandOptions{})
	if err != nil {
		return fmt.Errorf("docker compose down failed for project %s: %v. Output: %s", projectName, err, downOutput)
	}

	log.Debugf("Successfully stopped compose project %s", projectName)
	return nil
}

// ComposePurgeProject runs compose down with volumes/images removal for an explicit -p name.
func ComposePurgeProject(projectName string) error {
	log.Debugf("Purging docker compose project %s", projectName)

	purgeOutput, err := runRuntimeCommand("docker", []string{"compose", "-p", projectName,
		"down", "--remove-orphans", "--volumes", "--rmi", "all"}, runtimeCommandOptions{})
	if err != nil {
		return fmt.Errorf("docker compose purge failed for project %s: %v. Output: %s", projectName, err, purgeOutput)
	}

	log.Debugf("Successfully purged compose project %s", projectName)
	return nil
}

func cleanupFailedComposeDeployment(projectName string, deploymentErr error) error {
	if cleanupErr := ComposeDownProject(projectName); cleanupErr != nil {
		return errors.Join(deploymentErr, fmt.Errorf("clean up failed compose deployment: %w", cleanupErr))
	}
	return deploymentErr
}
