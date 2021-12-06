package cr

import (
	"bytes"
	"context"
	"io"

	"github.com/containerd/containerd"
)

type ContainerdClient struct {
	client *containerd.Client
}

func NewContainerdClient() (Runtime, error) {
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return &ContainerdClient{
		client: client,
	}, nil
}

func (self *ContainerdClient) ContainerList(ctx context.Context, options ContainerListOptions) ([]Container, error) {
	var filters string
	list, err := self.client.Containers(ctx, filters)
	if err != nil {
		return []Container{}, err
	}

	containers := make([]Container, len(list))
	for index, container := range list {
		info, _ := container.Info(ctx)
		// labels, _ := container.Labels(ctx)
		containers[index] = Container{
			ID:      info.ID,
			Image:   info.Image,
			Created: info.CreatedAt.Local().Unix(),
		}
	}
	return containers, err
}

func (self *ContainerdClient) ContainerStop(ctx context.Context, containerId string) error {
	// TODO: kill all running tasks
	return nil
}

func (self *ContainerdClient) ContainerRemove(ctx context.Context, containerId string, options ContainerRemoveOptions) error {
	container, err := self.client.LoadContainer(ctx, containerId)
	if err != nil {
		return err
	}

	err = container.Delete(ctx, containerd.WithSnapshotCleanup)
	return err
}

func (self *ContainerdClient) ContainerCreate(ctx context.Context, config *CreateContainerConfig, containerName string) (string, error) {
	// TODO: add config options
	container, err := self.client.NewContainer(ctx, containerName)
	if err != nil {
		return "", nil
	}

	return container.ID(), nil
}

func (self *ContainerdClient) ContainerStart(ctx context.Context, containerId string) error {
	// TODO: start a task
	return nil
}

func (self *ContainerdClient) ContainerLogs(ctx context.Context, containerId string, options ContainerLogsOptions) (string, error) {
	// TODO: persist IO from container task and output logs here
	return "", nil
}

func (self *ContainerdClient) ContainerCommit(ctx context.Context, containerId string) (string, error) {
	container, err := self.client.LoadContainer(ctx, containerId)
	if err != nil {
		return "", err
	}

	if err = container.Update(ctx); err != nil {
		return "", nil
	}
	return container.ID(), nil

}

// TODO: add buildkit and handle image related operations
func (self *ContainerdClient) ImageBuild(ctx context.Context, builderContext io.Reader, options ImageBuildOptions) (*bytes.Buffer, error) {
	return nil, nil
}

func (self *ContainerdClient) ImageRemove(ctx context.Context, imageId string, options ImageRemoveOptions) error {
	return nil
}

func (self *ContainerdClient) ImageInspect(ctx context.Context, imageId string) (Image, error) {
	return Image{}, nil
}

func (self *ContainerdClient) ImageList(ctx context.Context, options ImageListOptions) ([]Image, error) {
	return []Image{}, nil
}
