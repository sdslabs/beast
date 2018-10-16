package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

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

	// Try to stop using default timeout of docker
	err = cli.ContainerStop(context.Background(), containerId, nil)
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

func CreateContainerFromImage(portsList []uint32, mountsMap map[string]string, imageId string, containerName string) (string, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}
	portSet := make(nat.PortSet)
	portMap := make(nat.PortMap)

	for _, port := range portsList {
		natPort, err := nat.NewPort("tcp", strconv.Itoa(int(port)))
		if err != nil {
			return "", fmt.Errorf("Error while creating new port from port %d", port)
		}

		portSet[natPort] = struct{}{}

		portMap[natPort] = []nat.PortBinding{{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(int(port)),
		}}
	}

	config := &container.Config{
		Image:        imageId,
		ExposedPorts: portSet,
	}

	var mountBindings []mount.Mount
	for src, dest := range mountsMap {
		mnt := mount.Mount{
			Type:   mount.TypeBind,
			Source: src,
			Target: dest,
		}

		mountBindings = append(mountBindings, mnt)
	}

	hostConfig := &container.HostConfig{
		PortBindings: portMap,
		Mounts:       mountBindings,
	}

	createResp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, containerName)
	if err != nil {
		log.Error("Error while creating the container with name %s", containerName)
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
