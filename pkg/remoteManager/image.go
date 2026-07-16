package remoteManager

import (
	"errors"

	"github.com/sdslabs/beastv4/core/config"
)

// Remove image from remote server.
func RemoveImageRemote(imageId string, server config.AvailableServer) error {
	_, err := RunArgsOnServer(server, "docker", "rmi", imageId)
	return err
}

// Check for existence on image on remote server
func CheckIfImageExistsOnRemote(imageId string, server config.AvailableServer) (bool, error) {
	output, err := RunArgsOnServer(server, "docker", "inspect", "--format", "{{.ID}}", imageId)
	if err != nil {
		var commandError *RemoteCommandError
		if errors.As(err, &commandError) && commandError.ExitStatus == 1 {
			return false, nil
		}
		return false, err
	}
	return output != "", nil
}
