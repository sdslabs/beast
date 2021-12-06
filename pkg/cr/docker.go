package cr

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
)

type DockerClient struct {
	client *client.Client
}

func NewDockerClient() (Runtime, error) {
	client, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		client: client,
	}, nil
}

func (self *DockerClient) ContainerList(ctx context.Context, options ContainerListOptions) ([]Container, error) {
	filterArgs := filters.NewArgs()
	for key, val := range options.Filters {
		filterArgs.Add(key, val)
	}

	list, err := self.client.ContainerList(ctx, types.ContainerListOptions{
		All:     options.All,
		Filters: filterArgs,
	})

	if err != nil {
		return []Container{}, err
	}

	containers := make([]Container, len(list))
	for index, container := range list {
		containers[index] = Container{
			ID:      container.ID,
			Names:   container.Names,
			Image:   container.Image,
			ImageID: container.ImageID,
			Command: container.Command,
			Created: container.Created,
			State:   container.State,
			Status:  container.Status,
		}
	}
	return containers, err
}

func (self *DockerClient) ContainerStop(ctx context.Context, containerId string) error {
	err := self.client.ContainerStop(ctx, containerId, &defaults.DefaultDockerStopTimeout)
	return err
}

func (self *DockerClient) ContainerRemove(ctx context.Context, containerId string, options ContainerRemoveOptions) error {
	err := self.client.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{
		RemoveVolumes: options.RemoveVolumes,
		RemoveLinks:   options.RemoveLinks,
		Force:         options.Force,
	})
	return err
}

func (self *DockerClient) ContainerCreate(ctx context.Context, containerConfig *CreateContainerConfig, containerName string) (string, error) {
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

	response, err := self.client.ContainerCreate(ctx, config, hostConfig, nil, containerName)
	if err != nil {
		log.Error("Error while creating the container with name %s", containerName)
		return "", err
	}
	return response.ID, nil
}

func (self *DockerClient) ContainerStart(ctx context.Context, containerId string) error {
	if err := self.client.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		return err
	}
	return nil
}

func (self *DockerClient) ContainerLogs(ctx context.Context, containerId string, options ContainerLogsOptions) (string, error) {
	logChan, err := self.client.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{
		ShowStdout: options.ShowStdout,
		ShowStderr: options.ShowStderr,
		Details:    options.Details,
	})
	if err != nil {
		return "", err
	}
	defer logChan.Close()

	logs, _ := ioutil.ReadAll(logChan)
	return string(logs), nil
}

func (self *DockerClient) ContainerCommit(ctx context.Context, containerId string) (string, error) {
	response, err := self.client.ContainerCommit(ctx, containerId, types.ContainerCommitOptions{})
	if err != nil {
		return "", err
	}
	return response.ID, nil
}

func (self *DockerClient) ImageBuild(ctx context.Context, builderContext io.Reader, options ImageBuildOptions) (*bytes.Buffer, error) {
	buildOptions := types.ImageBuildOptions{
		Tags:       options.Tags,
		Remove:     options.Remove,
		Dockerfile: options.Dockerfile,
		NoCache:    options.NoCache,
	}

	imageBuildResp, err := self.client.ImageBuild(ctx, builderContext, buildOptions)
	if err != nil {
		return nil, err
	}
	defer imageBuildResp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(imageBuildResp.Body)

	return buf, nil
}

func (self *DockerClient) ImageRemove(ctx context.Context, imageId string, options ImageRemoveOptions) error {
	_, err := self.client.ImageRemove(context.Background(), imageId, types.ImageRemoveOptions{
		Force:         options.Force,
		PruneChildren: options.PruneChildren,
	})
	return err
}

func (self *DockerClient) ImageInspect(ctx context.Context, imageId string) (Image, error) {
	inspectVal, _, err := self.client.ImageInspectWithRaw(ctx, imageId)
	if err != nil {
		return Image{}, err
	}

	image := Image{
		ID: inspectVal.ID,
	}
	return image, nil
}

func (self *DockerClient) ImageList(ctx context.Context, options ImageListOptions) ([]Image, error) {
	filterArgs := filters.NewArgs()
	for key, val := range options.Filters {
		filterArgs.Add(key, val)
	}

	list, err := self.client.ImageList(ctx, types.ImageListOptions{
		All:     options.All,
		Filters: filterArgs,
	})

	if err != nil {
		return []Image{}, err
	}

	images := make([]Image, len(list))
	for index, image := range list {
		images[index] = Image{
			Containers:  image.Containers,
			Created:     image.Created,
			ID:          image.ID,
			Labels:      image.Labels,
			ParentID:    image.ParentID,
			RepoDigests: image.RepoDigests,
			RepoTags:    image.RepoTags,
			SharedSize:  image.SharedSize,
			Size:        image.Size,
			VirtualSize: image.VirtualSize,
		}
	}
	return images, err

}
