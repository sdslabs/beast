package cr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sdslabs/beastv4/core"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

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

func CheckIfImageExists(imageId string) (bool, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return false, err
	}

	inspectVal, _, err := cli.ImageInspectWithRaw(ctx, imageId)
	if err != nil {
		return false, err
	}

	if inspectVal.ID != "" {
		return true, nil
	}

	return false, nil
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

func BuildImageFromTarContext(challengeName, challengeTag, tarContextPath, dockerCtxFile string, noCache bool) (*bytes.Buffer, string, error) {
	ctx := context.Background()
	builderContext, err := os.Open(tarContextPath)
	if err != nil {
		return nil, "", fmt.Errorf("error while opening staged file :: %s", tarContextPath)
	}
	defer builderContext.Close()

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{challengeTag},
		Remove:     true,
		Dockerfile: dockerCtxFile,
		NoCache:    noCache,
	}

	dockerClient, err := client.NewEnvClient()
	if err != nil {
		return nil, "", fmt.Errorf("error while creating a docker client for beast: %s", err)
	}

	log.Debug("Image build in process")
	imageBuildResp, err := dockerClient.ImageBuild(ctx, builderContext, buildOptions)
	if err != nil {
		return nil, "", fmt.Errorf("an error while build image for challenge %s :: %s", challengeName, err)
	}
	defer imageBuildResp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(imageBuildResp.Body)

	images, err := SearchImageByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", challengeTag)})
	if len(images) > 0 {
		log.Infof("Image ID for the image built is : %s", images[0].ID[7:])
		return buf, images[0].ID[7:], nil
	}

	return buf, "", err
}

// TODO: find a better way to build images from docker-compose instead of cmd running
func BuildImagesFromCompose(challengeName, challengeTag, stagedPath, ComposeFile string, noCache bool) (*bytes.Buffer, error) {
	extractPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, challengeName)

	extractCmd := fmt.Sprintf("mkdir -p %s && tar -xf %s -C %s", extractPath, stagedPath, extractPath)
	err := exec.Command("bash", "-c", extractCmd).Run()
	if err != nil {
		return nil, fmt.Errorf("error while extracting tar file %s to %s: %v", stagedPath, extractPath, err)
	}
	chngDir := fmt.Sprintf("cd %s", extractPath)
	cmdArgs := []string{"compose", "build"}
	if noCache {
		cmdArgs = append(cmdArgs, "--no-cache")
	}
	composeCmd := fmt.Sprintf("%s && docker %s", chngDir, strings.Join(cmdArgs, " "))
	log.Debugf("Building image for challenge %s with tag %s", challengeName, challengeTag)
	log.Debugf("Running the command: docker %v", cmdArgs)

	cmd := exec.Command("bash", "-c", composeCmd)
	cmd.Dir = extractPath
	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &outBuffer

	if err := cmd.Run(); err != nil {
		return &outBuffer, fmt.Errorf("error while building image for challenge %s with tag %s: %v", challengeName, challengeTag, err)
	}
	return &outBuffer, nil
}
