package remoteManager

import (
	"fmt"
	"strings"

	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/pkg/cr"
)

func RunCheckerContainerRemote(server config.AvailableServer, image, networkName, volumeName, instanceID string, command []string) (cr.ExecResult, error) {
	dockerArgs := []string{
		"docker", "run", "--rm",
		"--network", networkName,
		"--mount", fmt.Sprintf("type=volume,source=%s,target=/challenge,readonly", volumeName),
		"--label", "beast.checker=true",
		"--label", fmt.Sprintf("beast.instance.id=%s", instanceID),
		image,
	}
	dockerArgs = append(dockerArgs, command...)

	quotedArgs := make([]string, len(dockerArgs))
	for i, arg := range dockerArgs {
		quotedArgs[i] = shellQuote(arg)
	}
	return RunCommandOnServerWithExit(server, strings.Join(quotedArgs, " "))
}
