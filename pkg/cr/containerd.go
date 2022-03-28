package cr

import (
	"bytes"
	"context"
	"io"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/images"
)

var LogBuffers map[string]*cio.Creator
var TaskMaps map[string]*containerd.Task

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
	// TODO: handle filters
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
	task := *TaskMaps[containerId]

	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return err
	}

	_, err := task.Delete(ctx)
	if err != nil {
		return err
	}
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
	container, err := self.client.LoadContainer(ctx, containerId)
	if err != nil {
		return err
	}

	cio := cio.NewCreator(cio.WithStdio)
	task, err := container.NewTask(ctx, cio)
	if err != nil {
		return err
	}

	TaskMaps[containerId] = &task
	LogBuffers[containerId] = &cio

	_, err = task.Wait(ctx)
	if err != nil {
		return err
	}

	if err := task.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (self *ContainerdClient) ContainerLogs(ctx context.Context, containerId string, options ContainerLogsOptions) (string, error) {
	cio := *LogBuffers[containerId]
	io, err := cio(containerId)
	if err != nil {
		return "", nil
	}

	_ = io.Config()
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
	// var image images.Image

	// builder, err := client.New(ctx, "//TODO: buildkit daemon address", client.WithFailFast())
	// if err != nil {
	// 	return nil, err
	// }

	// ch := make(chan *client.SolveStatus)
	// _, err = builder.Build(ctx, *solveOpt, "", dockerfile.Build, ch)

	// imageStore := self.client.ImageService()
	// image, err = imageStore.Create(ctx, image)
	// return nil, err
	return nil, nil
}

func (self *ContainerdClient) ImageRemove(ctx context.Context, imageId string, options ImageRemoveOptions) error {
	imageStore := self.client.ImageService()
	err := imageStore.Delete(ctx, imageId, images.SynchronousDelete())
	return err
}

func (self *ContainerdClient) ImageInspect(ctx context.Context, imageId string) (Image, error) {
	imageStore := self.client.ImageService()
	inspectVal, err := imageStore.Get(ctx, imageId)
	if err != nil {
		return Image{}, err
	}

	image := Image{
		ID: inspectVal.Name,
	}
	return image, nil
}

func (self *ContainerdClient) ImageList(ctx context.Context, options ImageListOptions) ([]Image, error) {
	var filters string
	imageStore := self.client.ImageService()
	list, err := imageStore.List(ctx, filters)
	if err != nil {
		return []Image{}, err
	}

	images := make([]Image, len(list))
	for index, image := range list {
		images[index] = Image{
			Created: image.CreatedAt.Unix(),
			ID:      image.Name,
			Labels:  image.Labels,
			Size:    image.Target.Size,
		}
	}
	return images, err
}
