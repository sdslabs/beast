package cr

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func RunCheckerContainer(image, networkName, volumeName, instanceID string, command []string) (ExecResult, error) {
	result := ExecResult{ExitCode: 1}
	args := []string{
		"run", "--rm",
		"--network", networkName,
		"--mount", fmt.Sprintf("type=volume,source=%s,target=/challenge,readonly", volumeName),
		"--label", "beast.checker=true",
		"--label", fmt.Sprintf("beast.instance.id=%s", instanceID),
		image,
	}
	args = append(args, command...)

	cmd := exec.Command("docker", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	result.Output = strings.TrimSpace(output.String())
	if err == nil {
		result.ExitCode = 0
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, err
}
