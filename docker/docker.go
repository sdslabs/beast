package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func SearchContainerByFilter(filterMap map[string]string) ([]types.Container, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return []types.Container{}, err
	}

	filterArgs := filters.Args{}
	for key, val := range filterMap {
		filterArgs.Add(key, val)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		All:     false,
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
	err := cli.ContainerStop(context.Background(), containerId, nil)
	if err != nil {
		return err
	}

	err := cli.ContainerRemove(context.Background(), containerId, types.ContainerRemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   true,
		Force:         true,
	})

	return err
}

func RemoveImage(imageId string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	err := cli.ImageRemove(context.Background(), imageId, types.ImageRemoveOptions{
		Force:         false,
		PruneChildren: true,
	})

	return err
}
