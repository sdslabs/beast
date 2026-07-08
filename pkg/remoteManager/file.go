package remoteManager

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"os/exec"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

func ValidateFileRemoteExists(server config.AvailableServer, stagedChallengePath string) error {
	output, err := RunCommandOnServer(server, fmt.Sprintf("test -e %s && echo exists || echo not exists", shellQuote(stagedChallengePath)))
	if err != nil {
		log.Errorf("Error while checking file existence: %s\n", err)
		return err
	}
	log.Printf("Output: %s\n", output)
	if strings.TrimSpace(output) == "exists" {
		return nil
	} else {
		return fmt.Errorf("path %s does not exist in remote server %s", stagedChallengePath, server.Host)
	}
}

// Rsync any file to other servers for chall deployment
func RsyncFileToServer(server config.AvailableServer, localFilePath, remoteFilePath string) error {
	err := utils.ValidateDirExists(localFilePath)
	if err != nil {
		return fmt.Errorf("file %s does not exist: %s", localFilePath, err)
	}
	fmt.Printf("Rsyncing %s to %s:%s\n", localFilePath, server.Host, remoteFilePath)
	cmd := exec.Command("rsync", "-avz",
		"-e", fmt.Sprintf("ssh -i %s", server.SSHKeyPath),
		localFilePath,
		fmt.Sprintf("%s@%s:%s", server.Username, server.Host, remoteFilePath))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	fmt.Println("Result: " + out.String())
	return nil
}

func StageChallRemote(server config.AvailableServer, challenge database.Challenge) error {
	client, err := CreateSSHClient(server)
	if err != nil {
		return fmt.Errorf("SSH connection to %s failed: %s", server.Host, err)
	}
	defer client.Close()

	stagingDirPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	stagingRemoteDirPath := filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	// err = RunCommandOnServer(server, fmt.Sprintf("mkdir -p %s/%s", remoteStagingDir, challenge.Name))
	// if err != nil {
	// 	return fmt.Errorf("failed to create directory: %s", err)
	// }

	// Rsync the challenge files to the server
	err = RsyncFileToServer(server, fmt.Sprintf("%s/%s", stagingDirPath, challenge.Name), stagingRemoteDirPath)
	if err != nil {
		return fmt.Errorf("failed to rsync challenge files: %s", err)
	}

	return nil
}

// BuildImageFromTarContextRemote builds a Docker image from the tar context on the remote server.
func BuildImageFromTarContextRemote(challengeName string, imageTag string, stagedDir string, server config.AvailableServer) ([]byte, string, error) {
	remoteExtractPath := filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, challengeName)
	_, err := RunCommandOnServer(server, fmt.Sprintf("mkdir -p %s && tar -xf %s -C %s", shellQuote(remoteExtractPath), shellQuote(stagedDir), shellQuote(remoteExtractPath)))
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to extract tar: %s", err)
	}
	projectName := utils.ProjectNameNotInstanced(challengeName)
	dockerBuildCmd := fmt.Sprintf("cd %s && %s",
		shellQuote(remoteExtractPath),
		shellJoin(
			"docker", "build",
			"-t", imageTag,
			"--label", "beast.challenge="+challengeName,
			"--label", "com.sdslabs.beast.project="+projectName,
			"--label", "com.docker.compose.project="+projectName,
			".",
		),
	)
	output, err := RunCommandOnServer(server, dockerBuildCmd)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to build docker image: %s\nOutput: %s", err, output)
	}
	getImageIDCmd := shellJoin("docker", "image", "inspect", imageTag, "--format", "{{.ID}}")
	imageID, err := RunCommandOnServer(server, getImageIDCmd)
	if err != nil {
		return []byte(output), "", fmt.Errorf("failed to retrieve Docker image ID: %w", err)
	}
	if imageID == "" {
		return []byte{}, "", fmt.Errorf("failed to retrieve Docker image ID")
	}
	return []byte(output), strings.TrimPrefix(strings.TrimSpace(imageID), "sha256:"), nil
}

func BuildImagesFromComposeRemote(challengeName, imageTag, stagedDir string, server config.AvailableServer, noCache bool) ([]byte, error) {
	remoteExtractPath := filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, challengeName)
	_, err := RunCommandOnServer(server, fmt.Sprintf("mkdir -p %s && tar -xf %s -C %s", shellQuote(remoteExtractPath), shellQuote(stagedDir), shellQuote(remoteExtractPath)))
	if err != nil {
		return []byte{}, fmt.Errorf("failed to extract tar: %s", err)
	}
	cmdArgs := []string{"docker", "compose", "build"}
	if noCache {
		cmdArgs = append(cmdArgs, "--no-cache")
	}
	// Note: docker compose build does not support --label flag
	// Labels are automatically added to containers during 'docker compose up -p <project>'
	dockerComposeBuildCmd := fmt.Sprintf("cd %s && %s", shellQuote(remoteExtractPath), shellJoin(cmdArgs...))

	// Execute the command on the remote server
	output, err := RunCommandOnServer(server, dockerComposeBuildCmd)
	if err != nil {
		return []byte(output), fmt.Errorf("failed to build docker compose images remotely: %s\nOutput: %s", err, output)
	}

	return []byte(output), nil
}

func BuildSadServersCheckerImageRemote(challengeName, stagedTarPath string, server config.AvailableServer, noCache bool) ([]byte, string, string, error) {
	remoteExtractPath := filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, challengeName)
	checkerDockerfile := filepath.Join(remoteExtractPath, "Dockerfile.checker")
	imageRef := cr.SadServersCheckerImageRef(challengeName)

	extractCommand := fmt.Sprintf(
		"mkdir -p %s && tar -xf %s -C %s && mkdir -p %s",
		shellQuote(remoteExtractPath),
		shellQuote(stagedTarPath),
		shellQuote(remoteExtractPath),
		shellQuote(filepath.Join(remoteExtractPath, "checker")),
	)
	if output, err := RunCommandOnServer(server, extractCommand); err != nil {
		return []byte(output), "", imageRef, fmt.Errorf("failed to extract sadservers checker context: %s\nOutput: %s", err, output)
	}

	writeDockerfileCommand := fmt.Sprintf(
		"printf %%s %s > %s",
		shellQuote(sadServersCheckerDockerfile()),
		shellQuote(checkerDockerfile),
	)
	if output, err := RunCommandOnServer(server, writeDockerfileCommand); err != nil {
		return []byte(output), "", imageRef, fmt.Errorf("failed to write sadservers checker Dockerfile: %s\nOutput: %s", err, output)
	}

	noCacheArg := ""
	if noCache {
		noCacheArg = " --no-cache"
	}
	projectName := utils.ProjectNameNotInstanced(challengeName)
	buildCommand := fmt.Sprintf(
		"cd %s && docker build%s -f %s -t %s --label %s --label %s --label %s --label %s .",
		shellQuote(remoteExtractPath),
		noCacheArg,
		shellQuote(checkerDockerfile),
		shellQuote(imageRef),
		shellQuote("beast.checker=true"),
		shellQuote("beast.checker.mode=sadservers"),
		shellQuote("beast.challenge="+challengeName),
		shellQuote("com.sdslabs.beast.project="+projectName),
	)
	output, err := RunCommandOnServer(server, buildCommand)
	if err != nil {
		return []byte(output), "", imageRef, fmt.Errorf("failed to build sadservers checker image remotely: %s\nOutput: %s", err, output)
	}

	imageID, err := RunCommandOnServer(server, shellJoin("docker", "image", "inspect", imageRef, "--format", "{{.ID}}"))
	if err != nil {
		return []byte(output), "", imageRef, fmt.Errorf("failed to inspect sadservers checker image: %s", err)
	}

	return []byte(output), strings.TrimPrefix(strings.TrimSpace(imageID), "sha256:"), imageRef, nil
}

func sadServersCheckerDockerfile() string {
	return `FROM ubuntu:24.04
WORKDIR /checker
COPY check.sh /checker/check.sh
COPY checker/ /checker/
RUN chmod 0555 /checker/check.sh
ENTRYPOINT ["/checker/check.sh"]
`
}
