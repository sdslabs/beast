package cr

import (
	"bytes"
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

func DeployContainerFromCompose(challengeName, stagedPath string) error {
	extractDir := filepath.Join(stagedPath, challengeName)
	log.Debugf("Deploying challenge %s using docker-compose at %s", challengeName,extractDir )
	upCmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && docker compose up -d", extractDir))
	var upOutput bytes.Buffer
	upCmd.Stdout = &upOutput
	upCmd.Stderr = &upOutput

	if err := upCmd.Run(); err != nil {
		log.Errorf("docker-compose up failed for challenge %s. Output:\n%s", challengeName, upOutput.String())
		return fmt.Errorf("error while running docker compose up: %v", err)
	}

	var psOutput bytes.Buffer
	checkCmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && docker compose ps", extractDir))
	checkCmd.Stdout = &psOutput
	checkCmd.Stderr = &psOutput
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("error checking container status after compose up for challenge %s. Output:\n%s", challengeName, psOutput.String())
	}
	if !strings.Contains(psOutput.String(), "Up") {
		return fmt.Errorf("container not running after compose up for challenge %s. Output:\n%s", challengeName, psOutput.String())
	}

	return nil
}

func ComposeDown(challengeName, stagedDir string) error {
	log.Debugf("Stopping challenge %s using docker-compose", challengeName)
	extractDir := filepath.Join(stagedDir, challengeName)
	downCmd := exec.Command("bash", "-c",fmt.Sprintf("cd %s && docker compose down", extractDir))

	var downOutput bytes.Buffer
	downCmd.Stdout = &downOutput
	downCmd.Stderr = &downOutput

	if err := downCmd.Run(); err != nil {
		log.Errorf("docker-compose down failed for challenge %s. Output:\n%s", challengeName, downOutput.String())
		return fmt.Errorf("error while running docker compose down: %v", err)
	}

	return nil
}

func ComposePurge(challengeName, stagedDir string) error {
	log.Debugf("Purging challenge %s using docker-compose", challengeName)
	extractDir := filepath.Join(stagedDir, challengeName)
	purgeCmd := exec.Command("bash","-c",fmt.Sprintf("cd %s && docker compose down --remove-orphans --volumes --rmi all", extractDir))

	var purgeOutput bytes.Buffer
	purgeCmd.Stdout = &purgeOutput
	purgeCmd.Stderr = &purgeOutput

	if err := purgeCmd.Run(); err != nil {
		log.Errorf("docker-compose purge failed for challenge %s. Output:\n%s", challengeName, purgeOutput.String())
		return fmt.Errorf("error while running docker compose purge: %v", err)
	}

	return nil
}
