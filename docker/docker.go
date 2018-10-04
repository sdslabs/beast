package docker

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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
	err = cli.ContainerStop(context.Background(), containerId, nil)
	if err != nil {
		return err
	}

	err = cli.ContainerRemove(context.Background(), containerId, types.ContainerRemoveOptions{
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

	_, err = cli.ImageRemove(context.Background(), imageId, types.ImageRemoveOptions{
		Force:         false,
		PruneChildren: true,
	})

	return err
}

func SearchImageByFilter(filterMap map[string]string) ([]types.ImageSummary, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return []types.ImageSummary{}, err
	}

	filterArgs := filters.NewArgs()
	for key, val := range filterMap {
		filterArgs.Add(key, val)
	}

	images, err := cli.ImageList(context.Background(), types.ImageListOptions{
		All:     false,
		Filters: filterArgs,
	})

	return images, err
}

func BuildImageFromTarContext(challengeName, tarContextPath string) (*bytes.Buffer, string, error) {
	builderContext, err := os.Open(tarContextPath)
	if err != nil {
		return nil, "", fmt.Errorf("Error while opening staged file :: %s", tarContextPath)
	}
	defer builderContext.Close()

	buildOptions := types.ImageBuildOptions{
		Tags:   []string{challengeName},
		Remove: true,
	}

	dockerClient, err := client.NewEnvClient()
	if err != nil {
		return nil, "", fmt.Errorf("Error while creating a docker client for beast: %s", err)
	}

	log.Debug("Image build in process")
	imageBuildResp, err := dockerClient.ImageBuild(context.Background(), builderContext, buildOptions)
	if err != nil {
		return nil, "", fmt.Errorf("An error while build image for challenge %s :: %s", challengeName, err)
	}
	defer imageBuildResp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(imageBuildResp.Body)

	images, err := SearchImageByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", challengeName)})
	if len(images) > 0 {
		log.Infof("Image ID for the image built is : %s", images[0].ID[7:])
		return buf, images[0].ID[7:], nil
	}

	return buf, "", err
}

func CreateContainerFromImage(portsList []uint32, imageId string, challengeName string) (string, error) {
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

	hostConfig := &container.HostConfig{
		PortBindings: portMap,
	}

	createResp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, challengeName)
	if err != nil {
		log.Error("Error while creating the container with name %s", challengeName)
		return "", err
	}

	containerId := createResp.ID
	if len(createResp.Warnings) > 0 {
		log.Warnf("Warnings while creating the container : %s", createResp.Warnings)
	}

	if err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		log.Error("Error while starting the container")
	}

	return containerId, err
}
