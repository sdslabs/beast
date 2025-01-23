package cr

import (
	"fmt"
	"io/ioutil"
	"strconv"

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
	containerName := containerConfig.ContainerName
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
			return "", fmt.Errorf("Error while creating new port from port %d", portMapping.ContainerPort)
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
		PidsLimit: containerConfig.PidsLimit,
	}

	hostConfig := &container.HostConfig{
		PortBindings: portMap,
		Mounts:       mountBindings,
		NetworkMode:  container.NetworkMode(containerConfig.ContainerNetwork),
		Resources:    resources,
	}

	createResp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, containerName)
	if err != nil {
		log.Errorf("Error while creating the container with name %s", containerName)
		return "", err
	}

	containerId := createResp.ID
	if len(createResp.Warnings) > 0 {
		log.Warnf("Warnings while creating the container : %s", createResp.Warnings)
	}

	if err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		log.Error("Error while starting the container : %s", err)
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
