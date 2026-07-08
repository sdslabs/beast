package remoteManager

import (
	"os/exec"

	"github.com/sdslabs/beastv4/core/config"
)

// Remove image from remote server.
func RemoveImageRemote(imageId string, server config.AvailableServer) error {
	command := shellJoin("docker", "rmi", imageId)
	_, err := RunCommandOnServer(server, command)
	return err
}

// Check for existence on image on remote server
func CheckIfImageExistsOnRemote(imageId string, server config.AvailableServer) (bool, error) {
	command := shellJoin("docker", "inspect", "--format={{.ID}}", imageId)
	output, err := RunCommandOnServer(server, command)
	if err != nil {
		exitError, success := err.(*exec.ExitError)
		if success && exitError.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return output != "", nil
}
