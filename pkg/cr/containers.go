package cr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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
	utils "github.com/sdslabs/beastv4/utils"

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

	labels := map[string]string{
		"beast.challenge":             containerConfig.ChallengeName,
		"com.sdslabs.beast.project":   utils.GetProjectName(containerConfig.ChallengeName),
		"com.docker.compose.project":  utils.GetProjectName(containerConfig.ChallengeName),
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

func DeployContainerFromCompose(challengeName string, projectBase string, stagedPath string, composeFileName string, ports map[string]uint32) (string, error) {
	extractDir := filepath.Join(stagedPath, challengeName)
	projectName := utils.GetProjectName(projectBase)
	composeFile := filepath.Join(extractDir, composeFileName)

	log.Debugf("Deploying challenge %s using docker compose with project name %s and file %s", challengeName, projectName, composeFileName)

	// Deploy with project name - Docker Compose automatically labels containers with
	// com.docker.compose.project=<projectName>
	upCmd := exec.Command("docker", "compose",
		"-f", composeFile,
		"-p", projectName,
		"up", "-d")

	environment := os.Environ()
	for variable, port := range ports {
		environment = append(environment, fmt.Sprintf("%s=%s", variable, strconv.FormatUint(uint64(port), 10)))
	}

	upCmd.Env = environment

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
	projectName := utils.GetProjectName(challengeName)

	downCmd := exec.Command("docker", "compose", "-p", projectName, "down")
	var downOutput bytes.Buffer
	downCmd.Stdout = &downOutput
	downCmd.Stderr = &downOutput

	if err := downCmd.Run(); err != nil {
		return fmt.Errorf("docker compose down failed for challenge %s: %v. Output: %s", challengeName, err, downOutput.String())
	}

	log.Debugf("Successfully stopped challenge %s", challengeName)
	return nil
}

func ComposePurge(challengeName, stagedDir string) error {
	log.Debugf("Purging challenge %s using docker compose", challengeName)
	projectName := utils.GetProjectName(challengeName)

	purgeCmd := exec.Command("docker", "compose", "-p", projectName,
		"down", "--remove-orphans", "--volumes", "--rmi", "all")

	var purgeOutput bytes.Buffer
	purgeCmd.Stdout = &purgeOutput
	purgeCmd.Stderr = &purgeOutput

	if err := purgeCmd.Run(); err != nil {
		return fmt.Errorf("docker compose purge failed for challenge %s: %v. Output: %s", challengeName, err, purgeOutput.String())
	}

	log.Debugf("Successfully purged challenge %s", challengeName)
	return nil
}
