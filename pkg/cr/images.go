package cr

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func RemoveImage(imageId string) error {
	cli, err := newDockerClient()
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
	cli, err := newDockerClient()
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
	cli, err := newDockerClient()
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

	cmdArgs := []string{"compose", "build"}
	if noCache {
		cmdArgs = append(cmdArgs, "--no-cache")
	}
	// Note: docker compose build does not support --label flag
	// Labels are automatically added to containers during 'docker compose up -p <project>'
	composeCmd := fmt.Sprintf("docker %s", strings.Join(cmdArgs, " "))
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

func SadServersCheckerImageRef(challengeName string) string {
	return fmt.Sprintf("beast-checker-%s:latest", utils.EncodeID(challengeName))
}

func BuildSadServersCheckerImageFromTarContext(challengeName, tarContextPath string, noCache bool) (*bytes.Buffer, string, string, error) {
	builderContext, err := buildSadServersCheckerContext(tarContextPath)
	if err != nil {
		return nil, "", "", err
	}

	imageRef := SadServersCheckerImageRef(challengeName)
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageRef},
		Remove:     true,
		Dockerfile: "Dockerfile",
		NoCache:    noCache,
		Labels: map[string]string{
			"beast.checker":             "true",
			"beast.checker.mode":        "sadservers",
			"beast.challenge":           challengeName,
			"com.sdslabs.beast.project": utils.ProjectNameNotInstanced(challengeName),
		},
	}

	dockerClient, err := newDockerClient()
	if err != nil {
		return nil, "", "", fmt.Errorf("error while creating a docker client for beast: %s", err)
	}

	imageBuildResp, err := dockerClient.ImageBuild(context.Background(), builderContext, buildOptions)
	if err != nil {
		return nil, "", "", fmt.Errorf("error while building sadservers checker image for challenge %s: %s", challengeName, err)
	}
	defer imageBuildResp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(imageBuildResp.Body)

	images, err := SearchImageByFilter(map[string]string{"reference": imageRef})
	if len(images) > 0 {
		return buf, strings.TrimPrefix(images[0].ID, "sha256:"), imageRef, nil
	}

	return buf, "", imageRef, err
}

func WriteSadServersCheckerContextArchive(tarContextPath, destinationPath string) error {
	builderContext, err := buildSadServersCheckerContext(tarContextPath)
	if err != nil {
		return err
	}
	if err := utils.CreateIfNotExistDir(filepath.Dir(destinationPath)); err != nil {
		return fmt.Errorf("failed to create checker context directory: %w", err)
	}

	destination, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create checker context archive %s: %w", destinationPath, err)
	}

	gzipWriter := gzip.NewWriter(destination)
	_, copyErr := io.Copy(gzipWriter, builderContext)
	closeGzipErr := gzipWriter.Close()
	closeFileErr := destination.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to write checker context archive %s: %w", destinationPath, copyErr)
	}
	if closeGzipErr != nil {
		return fmt.Errorf("failed to close checker context gzip archive %s: %w", destinationPath, closeGzipErr)
	}
	if closeFileErr != nil {
		return fmt.Errorf("failed to close checker context archive %s: %w", destinationPath, closeFileErr)
	}

	return nil
}

func buildSadServersCheckerContext(tarContextPath string) (io.Reader, error) {
	source, err := os.Open(tarContextPath)
	if err != nil {
		return nil, fmt.Errorf("error while opening staged checker context %s: %w", tarContextPath, err)
	}
	defer source.Close()

	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		return nil, fmt.Errorf("error while opening staged checker gzip context %s: %w", tarContextPath, err)
	}
	defer gzipReader.Close()

	sourceTar := tar.NewReader(gzipReader)
	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	defer writer.Close()

	if err := writeCheckerDockerfile(writer); err != nil {
		return nil, err
	}
	if err := writeTarDirectory(writer, "checker", 0755); err != nil {
		return nil, err
	}

	foundCheck := false
	for {
		header, err := sourceTar.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error while reading staged checker context: %w", err)
		}

		name := normalizeTarPath(header.Name)
		if name == "" {
			continue
		}

		includeCheck := name == core.SAD_CHECK_SCRIPT
		includeCheckerFile := strings.HasPrefix(name, "checker/")
		if !includeCheck && !includeCheckerFile {
			continue
		}

		if includeCheck && header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("sadservers check.sh must be a regular file")
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			continue
		}

		outHeader := *header
		outHeader.Name = name
		if includeCheck {
			foundCheck = true
			outHeader.Mode = 0555
		}
		if err := writer.WriteHeader(&outHeader); err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if _, err := io.Copy(writer, sourceTar); err != nil {
			return nil, err
		}
	}

	if !foundCheck {
		return nil, fmt.Errorf("sadservers challenges must provide root-level check.sh")
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func writeCheckerDockerfile(writer *tar.Writer) error {
	const dockerfile = `FROM ubuntu:24.04
WORKDIR /checker
COPY check.sh /checker/check.sh
COPY checker/ /checker/
RUN chmod 0555 /checker/check.sh
ENTRYPOINT ["/checker/check.sh"]
`
	header := &tar.Header{
		Name: "Dockerfile",
		Mode: 0644,
		Size: int64(len(dockerfile)),
	}
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	_, err := writer.Write([]byte(dockerfile))
	return err
}

func writeTarDirectory(writer *tar.Writer, name string, mode int64) error {
	return writer.WriteHeader(&tar.Header{
		Name:     name,
		Mode:     mode,
		Typeflag: tar.TypeDir,
	})
}

func normalizeTarPath(name string) string {
	name = filepath.ToSlash(strings.TrimLeft(name, "/"))
	name = strings.TrimPrefix(name, "./")
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return ""
	}
	return clean
}
