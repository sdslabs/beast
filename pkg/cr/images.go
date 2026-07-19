package cr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-units"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

const (
	maxBuildOutput = 4 << 20
	buildTimeout   = 30 * time.Minute
)

var errBuildOutputLimit = errors.New("build output exceeds 4 MiB limit")

type BuildLimits struct {
	CPUShares int64
	CPUs      float32
	Memory    int64
	Pids      int64
}

type boundedBuildOutput struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	exceeded bool
}

func (output *boundedBuildOutput) Write(data []byte) (int, error) {
	output.mu.Lock()
	defer output.mu.Unlock()
	remaining := maxBuildOutput - output.buffer.Len()
	if remaining <= 0 {
		output.exceeded = true
		return 0, errBuildOutputLimit
	}
	if len(data) > remaining {
		_, _ = output.buffer.Write(data[:remaining])
		output.exceeded = true
		return remaining, errBuildOutputLimit
	}
	return output.buffer.Write(data)
}

func (output *boundedBuildOutput) Buffer() *bytes.Buffer {
	output.mu.Lock()
	defer output.mu.Unlock()
	return bytes.NewBuffer(append([]byte(nil), output.buffer.Bytes()...))
}

func RemoveImage(imageId string) error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPIRequestTimeout)
	defer cancel()

	_, err = cli.ImageRemove(ctx, imageId, types.ImageRemoveOptions{
		Force:         false,
		PruneChildren: true,
	})

	return err
}

func CheckIfImageExists(imageId string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPIRequestTimeout)
	defer cancel()
	cli, err := newDockerClient()
	if err != nil {
		return false, err
	}
	defer cli.Close()

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
	cli, err := newDockerClient()
	if err != nil {
		return []types.ImageSummary{}, err
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), dockerAPIRequestTimeout)
	defer cancel()

	filterArgs := filters.NewArgs()
	for key, val := range filterMap {
		filterArgs.Add(key, val)
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{
		All:     false,
		Filters: filterArgs,
	})

	return images, err
}

func BuildImageFromTarContext(challengeName, challengeTag, tarContextPath, dockerCtxFile string, noCache bool, limits BuildLimits) (*bytes.Buffer, string, error) {
	if err := limits.Validate(); err != nil {
		return nil, "", fmt.Errorf("invalid build resource limits: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()
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
		CPUShares:  limits.CPUShares,
		CPUPeriod:  100000,
		CPUQuota:   CPUQuota(limits.CPUs),
		Memory:     limits.Memory,
		MemorySwap: limits.Memory,
		Ulimits: []*units.Ulimit{{
			Name: "nproc",
			Soft: limits.Pids,
			Hard: limits.Pids,
		}},
		Labels: map[string]string{
			"beast.challenge":            challengeName,
			"com.sdslabs.beast.project":  utils.ProjectNameNotInstanced(challengeName),
			"com.docker.compose.project": utils.ProjectNameNotInstanced(challengeName),
		},
	}

	dockerClient, err := newDockerClient()
	if err != nil {
		return nil, "", fmt.Errorf("error while creating a docker client for beast: %s", err)
	}
	defer dockerClient.Close()

	log.Debug("Image build in process")
	imageBuildResp, err := dockerClient.ImageBuild(ctx, builderContext, buildOptions)
	if err != nil {
		return nil, "", fmt.Errorf("an error while build image for challenge %s :: %s", challengeName, err)
	}
	defer imageBuildResp.Body.Close()

	buf := new(bytes.Buffer)
	written, err := io.Copy(buf, io.LimitReader(imageBuildResp.Body, maxBuildOutput+1))
	if err != nil {
		return buf, "", fmt.Errorf("read image build output: %w", err)
	}
	if written > maxBuildOutput {
		return buf, "", errBuildOutputLimit
	}
	if err := ctx.Err(); err != nil {
		return buf, "", fmt.Errorf("image build deadline: %w", err)
	}

	images, err := SearchImageByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", challengeTag)})
	if err != nil {
		return buf, "", fmt.Errorf("find built image: %w", err)
	}
	if len(images) > 0 {
		log.Infof("Image ID for the image built is : %s", images[0].ID[7:])
		return buf, images[0].ID[7:], nil
	}

	return buf, "", fmt.Errorf("Docker build completed without producing image %s:latest", challengeTag)
}

// TODO: find a better way to build images from docker-compose instead of cmd running
func BuildImagesFromCompose(challengeName, challengeTag, stagedPath, ComposeFile string, noCache bool) (*bytes.Buffer, error) {
	extractPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, challengeName)

	if err := os.MkdirAll(extractPath, 0750); err != nil {
		return nil, fmt.Errorf("create compose extraction directory %s: %w", extractPath, err)
	}
	err := exec.Command("tar", "-xf", stagedPath, "-C", extractPath).Run()
	if err != nil {
		return nil, fmt.Errorf("error while extracting tar file %s to %s: %v", stagedPath, extractPath, err)
	}

	cmdArgs := []string{"compose", "build"}
	if noCache {
		cmdArgs = append(cmdArgs, "--no-cache")
	}
	// Note: docker compose build does not support --label flag
	// Labels are automatically added to containers during 'docker compose up -p <project>'
	log.Debugf("Building image for challenge %s with tag %s", challengeName, challengeTag)
	log.Debugf("Running the command: docker %v", cmdArgs)

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Dir = extractPath
	output := &boundedBuildOutput{}
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		return output.Buffer(), fmt.Errorf("error while building image for challenge %s with tag %s: %v", challengeName, challengeTag, err)
	}
	if output.exceeded {
		return output.Buffer(), errBuildOutputLimit
	}
	if err := ctx.Err(); err != nil {
		return output.Buffer(), fmt.Errorf("Compose build deadline: %w", err)
	}
	return output.Buffer(), nil
}
