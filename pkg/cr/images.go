package cr

import (
	"bytes"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func RemoveImage(imageId string) error {
	cli, err := NewClient()
	if err != nil {
		return err
	}

	err = cli.ImageRemove(context.Background(), imageId, ImageRemoveOptions{
		Force:         false,
		PruneChildren: true,
	})

	return err
}

func CheckIfImageExists(imageId string) (bool, error) {
	ctx := context.Background()
	cli, err := NewClient()
	if err != nil {
		return false, err
	}

	inspectVal, err := cli.ImageInspect(ctx, imageId)
	if err != nil {
		return false, err
	}

	if inspectVal.ID != "" {
		return true, nil
	}

	return false, nil
}

func SearchImageByFilter(filterMap map[string]string) ([]Image, error) {
	cli, err := NewClient()
	if err != nil {
		return []Image{}, err
	}

	images, err := cli.ImageList(context.Background(), ImageListOptions{
		All:     false,
		Filters: filterMap,
	})

	return images, err
}

func BuildImageFromTarContext(challengeName, challengeTag, tarContextPath, dockerCtxFile string, noCache bool) (*bytes.Buffer, string, error) {
	builderContext, err := os.Open(tarContextPath)
	if err != nil {
		return nil, "", fmt.Errorf("Error while opening staged file :: %s", tarContextPath)
	}
	defer builderContext.Close()

	buildOptions := ImageBuildOptions{
		Tags:       []string{challengeTag},
		Remove:     true,
		Dockerfile: dockerCtxFile,
		NoCache:    noCache,
	}

	dockerClient, err := NewClient()
	if err != nil {
		return nil, "", fmt.Errorf("Error while creating a docker client for beast: %s", err)
	}

	log.Debug("Image build in process")
	imageBuildResp, err := dockerClient.ImageBuild(context.Background(), builderContext, buildOptions)
	if err != nil {
		return nil, "", fmt.Errorf("An error while build image for challenge %s :: %s", challengeName, err)
	}

	images, err := SearchImageByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", challengeTag)})
	if len(images) > 0 {
		log.Infof("Image ID for the image built is : %s", images[0].ID[7:])
		return imageBuildResp, images[0].ID[7:], nil
	}

	return imageBuildResp, "", err
}
