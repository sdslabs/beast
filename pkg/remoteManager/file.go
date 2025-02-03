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
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

func ValidateFileRemoteExists(server config.AvailableServer, stagedChallengePath string) error {
	output, err := RunCommandOnServer(server, fmt.Sprintf("test -e %s&&echo exists||echo not exists", stagedChallengePath))
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
	stagingRemoteDirPath := filepath.Join("~/.beast", core.BEAST_STAGING_DIR)
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
//
//	TODO: Sidecar's configuration left. Need to be added.
func BuildImageFromTarContextRemote(challengeName string, imageTag string, stagedDir string, server config.AvailableServer) ([]byte, string, error) {
	remoteExtractPath := filepath.Join("~/.beast", core.BEAST_STAGING_DIR, challengeName, challengeName)
	_, err := RunCommandOnServer(server, fmt.Sprintf("mkdir -p %s && tar -xf %s -C %s", remoteExtractPath, stagedDir, remoteExtractPath))
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to extract tar: %s", err)
	}
	dockerBuildCmd := fmt.Sprintf("cd %s && docker build -t %s .", remoteExtractPath, imageTag)
	output, err := RunCommandOnServer(server, dockerBuildCmd)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to build docker image: %s\nOutput: %s", err, output)
	}
	getImageIDCmd := fmt.Sprintf(
		"docker images --format '{{.Repository}} {{.ID}}' | grep %s | awk '{print $2}'",
		imageTag,
	)
	imageID, err := RunCommandOnServer(server, getImageIDCmd)
	if err != nil {
		log.Fatalf("Failed to retrieve Docker image ID: %v", err)
	}
	if imageID == "" {
		return []byte{}, "", fmt.Errorf("failed to retrieve Docker image ID")
	}
	return []byte(output), strings.TrimSpace(imageID), nil
}
