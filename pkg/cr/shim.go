package cr

import (
	"bytes"
	"context"
	"io"

	"github.com/docker/docker/client"
)

type Container struct {
	ID      string
	Names   []string
	Image   string
	ImageID string
	Command string
	Created int64
	State   string
	Status  string
}

type ContainerListOptions struct {
	Quiet   bool
	Size    bool
	All     bool
	Latest  bool
	Since   string
	Before  string
	Limit   int
	Filters map[string]string
}

type ContainerRemoveOptions struct {
	RemoveVolumes bool
	RemoveLinks   bool
	Force         bool
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

type ContainerLogsOptions struct {
	ShowStdout bool
	ShowStderr bool
	Details    bool
}

type Image struct {
	Containers  int64
	Created     int64
	ID          string
	Labels      map[string]string
	ParentID    string
	RepoDigests []string
	RepoTags    []string
	SharedSize  int64
	Size        int64
	VirtualSize int64
}

type ImageBuildOptions struct {
	Tags       []string
	Remove     bool
	Dockerfile string
	NoCache    bool
}

type ImageListOptions struct {
	All     bool
	Filters map[string]string
}

type ImageRemoveOptions struct {
	Force         bool
	PruneChildren bool
}

type Runtime interface {
	ContainerList(ctx context.Context, options ContainerListOptions) ([]Container, error)
	ContainerStop(ctx context.Context, containerId string) error
	ContainerRemove(ctx context.Context, containerId string, options ContainerRemoveOptions) error
	ContainerCreate(ctx context.Context, config *CreateContainerConfig, containerName string) (string, error)
	ContainerStart(ctx context.Context, containerId string) error
	ContainerLogs(ctx context.Context, containerId string, options ContainerLogsOptions) (string, error)
	ContainerCommit(ctx context.Context, containerId string) (string, error)
	ImageBuild(ctx context.Context, builderContext io.Reader, options ImageBuildOptions) (*bytes.Buffer, error)
	ImageRemove(ctx context.Context, imageId string, options ImageRemoveOptions) error
	ImageInspect(ctx context.Context, imageId string) (Image, error)
	ImageList(ctx context.Context, options ImageListOptions) ([]Image, error)
}

func NewClient() (Runtime, error) {
	client, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		client: client,
	}, nil
}
