package cr

import (
	"fmt"

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

func SearchContainerByFilter(filterMap map[string]string) ([]Container, error) {
	cli, err := NewClient()
	if err != nil {
		return []Container{}, err
	}

	containers, err := cli.ContainerList(context.Background(), ContainerListOptions{
		All:     true,
		Filters: filterMap,
	})

	return containers, err
}

func StopAndRemoveContainer(containerId string) error {
	cli, err := NewClient()
	if err != nil {
		return err
	}

	// Try to stop using default timeout we are using for beast
	err = cli.ContainerStop(context.Background(), containerId)
	if err != nil {
		return err
	}
	log.Debug("Stopped container with ID ", containerId)

	log.Debug("Removing container with ID ", containerId)
	err = cli.ContainerRemove(context.Background(), containerId, ContainerRemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   false,
		Force:         true,
	})

	return err
}

func CreateContainerFromImage(containerConfig *CreateContainerConfig) (string, error) {
	containerName := containerConfig.ContainerName
	ctx := context.Background()
	cli, err := NewClient()
	if err != nil {
		return "", err
	}

	containerId, err := cli.ContainerCreate(ctx, containerConfig, containerName)
	if err != nil {
		log.Error("Error while creating the container with name %s", containerName)
		return "", err
	}

	// containerId := createResp.ID
	// if len(createResp.Warnings) > 0 {
	// 	log.Warnf("Warnings while creating the container : %s", createResp.Warnings)
	// }

	if err := cli.ContainerStart(ctx, containerId); err != nil {
		log.Error("Error while starting the container : %s", err)
		return "", err
	}

	return containerId, nil
}

func GetContainerStdLogs(containerID string) (*Log, error) {
	cli, err := NewClient()
	if err != nil {
		return nil, err
	}

	stdoutlogs, err := cli.ContainerLogs(context.Background(), containerID, ContainerLogsOptions{
		ShowStdout: true,
		Details:    true,
	})
	if err != nil {
		return nil, err
	}

	stderrlogs, err := cli.ContainerLogs(context.Background(), containerID, ContainerLogsOptions{
		ShowStderr: true,
		Details:    true,
	})
	if err != nil {
		return nil, err
	}

	return &Log{Stdout: stdoutlogs, Stderr: stderrlogs}, nil
}

func ShowLiveContainerLogs(containerID string) {
	cli, err := NewClient()
	if err != nil {
		log.Error(err)
	}

	logs, err := cli.ContainerLogs(context.Background(), containerID, ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Details:    true,
	})
	if err != nil {
		log.Error(err)
	}

	fmt.Println(logs)
}

func CommitContainer(containerId string) (string, error) {
	ctx := context.Background()
	cli, err := NewClient()
	if err != nil {
		return "", err
	}

	commitResp, err := cli.ContainerCommit(ctx, containerId)
	if err != nil {
		return "", err
	}

	return commitResp, nil
}
