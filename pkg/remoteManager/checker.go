package remoteManager

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sdslabs/beastv4/core"
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

func RunSadServersCheckerContainerRemote(server config.AvailableServer, image, networkName, instanceID, challengeName string) (cr.ExecResult, error) {
	dockerArgs := []string{
		"timeout", strconv.Itoa(core.DEFAULT_CHECKER_TIMEOUT),
		"docker", "run", "--rm",
		"--pull", "never",
		"--network", networkName,
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--memory", "128m",
		"--pids-limit", "64",
		"--tmpfs", "/tmp:rw,nosuid,nodev,size=16m",
		"--env", fmt.Sprintf("BEAST_INSTANCE_ID=%s", instanceID),
		"--env", fmt.Sprintf("BEAST_CHALLENGE_NAME=%s", challengeName),
		"--env", "BEAST_TARGET_SERVICE=ssh",
		"--env", "BEAST_TARGET_HOST=ssh",
		"--env", fmt.Sprintf("BEAST_TARGET_PORT=%d", core.SSH_PORT),
		"--label", "beast.checker=true",
		"--label", "beast.checker.mode=sadservers",
		"--label", fmt.Sprintf("beast.instance.id=%s", instanceID),
		image,
	}

	quotedArgs := make([]string, len(dockerArgs))
	for i, arg := range dockerArgs {
		quotedArgs[i] = shellQuote(arg)
	}
	return RunCommandOnServerWithExit(server, strings.Join(quotedArgs, " "))
}
