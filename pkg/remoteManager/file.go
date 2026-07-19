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
)

func ValidateFileRemoteExists(server config.AvailableServer, stagedChallengePath string) error {
	_, err := RunArgsOnServer(server, "test", "-e", stagedChallengePath)
	if err != nil {
		return fmt.Errorf("path %s does not exist in remote server %s", stagedChallengePath, server.Host)
	}
	return nil
}

// Rsync any file to other servers for chall deployment
func RsyncFileToServer(server config.AvailableServer, localFilePath, remoteFilePath string) error {
	err := utils.ValidateDirExists(localFilePath)
	if err != nil {
		return fmt.Errorf("file %s does not exist: %s", localFilePath, err)
	}
	fmt.Printf("Rsyncing %s to %s:%s\n", localFilePath, server.Host, remoteFilePath)
	remoteShell := "ssh -i " + shellQuote(server.SSHKeyPath) +
		" -o " + shellQuote("UserKnownHostsFile="+server.KnownHostsFile) +
		" -o StrictHostKeyChecking=yes"
	cmd := exec.Command("rsync", "-az", "--protect-args",
		"-e", remoteShell, "--",
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
	// Rsync the challenge files to the server
	err = RsyncFileToServer(server, filepath.Join(stagingDirPath, challenge.Name), stagingRemoteDirPath)
	if err != nil {
		return fmt.Errorf("failed to rsync challenge files: %s", err)
	}

	return nil
}

// BuildImageFromTarContextRemote builds a Docker image from the tar context on the remote server.
func BuildImageFromTarContextRemote(challengeName string, imageTag string, stagedDir string, server config.AvailableServer, limits cr.BuildLimits) ([]byte, string, error) {
	remoteExtractPath := filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, challengeName)
	_, err := RunArgsOnServer(server, "mkdir", "-p", remoteExtractPath)
	if err == nil {
		_, err = RunArgsOnServer(server, "tar", "-xf", stagedDir, "-C", remoteExtractPath)
	}
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to extract tar: %s", err)
	}
	projectName := utils.ProjectNameNotInstanced(challengeName)
	output, err := RunArgsInDirOnServer(server, remoteExtractPath,
		"docker", "build", "-t", imageTag,
		"--cpu-shares", fmt.Sprintf("%d", limits.CPUShares),
		"--cpu-period", "100000",
		"--cpu-quota", fmt.Sprintf("%d", int64(limits.CPUs*100000)),
		"--memory", fmt.Sprintf("%d", limits.Memory),
		"--memory-swap", fmt.Sprintf("%d", limits.Memory),
		"--ulimit", fmt.Sprintf("nproc=%d:%d", limits.Pids, limits.Pids),
		"--label", "beast.challenge="+challengeName,
		"--label", "com.sdslabs.beast.project="+projectName,
		"--label", "com.docker.compose.project="+projectName, ".")
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to build docker image: %s\nOutput: %s", err, output)
	}
	imageID, err := RunArgsOnServer(server, "docker", "image", "inspect", "--format", "{{.Id}}", imageTag)
	if err != nil {
		return []byte(output), "", fmt.Errorf("retrieve Docker image ID: %w", err)
	}
	if imageID == "" {
		return []byte{}, "", fmt.Errorf("failed to retrieve Docker image ID")
	}
	return []byte(output), strings.TrimSpace(imageID), nil
}

func BuildImagesFromComposeRemote(challengeName, imageTag, stagedDir string, server config.AvailableServer, noCache bool) ([]byte, error) {
	remoteExtractPath := filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, challengeName)
	_, err := RunArgsOnServer(server, "mkdir", "-p", remoteExtractPath)
	if err == nil {
		_, err = RunArgsOnServer(server, "tar", "-xf", stagedDir, "-C", remoteExtractPath)
	}
	if err != nil {
		return []byte{}, fmt.Errorf("failed to extract tar: %s", err)
	}
	arguments := []string{"docker", "compose", "build"}
	if noCache {
		arguments = append(arguments, "--no-cache")
	}
	// Note: docker compose build does not support --label flag
	// Labels are automatically added to containers during 'docker compose up -p <project>'
	output, err := RunArgsInDirOnServer(server, remoteExtractPath, arguments...)
	if err != nil {
		return []byte(output), fmt.Errorf("failed to build docker compose images remotely: %s\nOutput: %s", err, output)
	}

	return []byte(output), nil
}
