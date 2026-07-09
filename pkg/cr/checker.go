package cr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core"
)

func RunCheckerContainer(image, networkName, volumeName, instanceID string, command []string) (ExecResult, error) {
	result := ExecResult{ExitCode: 1}
	args := []string{
		"run", "--rm",
		"--pull", "never",
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

func RunSadServersCheckerContainer(image, networkName, instanceID, challengeName string) (ExecResult, error) {
	result := ExecResult{ExitCode: 1}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(core.DEFAULT_CHECKER_TIMEOUT)*time.Second)
	defer cancel()

	checkerName := SadServersCheckerContainerName(instanceID)
	defer removeDockerContainer(checkerName)

	args := []string{
		"run", "--rm",
		"--name", checkerName,
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
		"--label", fmt.Sprintf("beast.checker.name=%s", checkerName),
		image,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	result.Output = strings.TrimSpace(output.String())
	if ctx.Err() == context.DeadlineExceeded {
		result.ExitCode = 124
		return result, nil
	}
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

func SadServersCheckerContainerName(instanceID string) string {
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	return fmt.Sprintf("beast-checker-%s-%s", dockerNameSegment(instanceID), suffix)
}

func dockerNameSegment(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	if builder.Len() == 0 {
		return "instance"
	}
	return builder.String()
}

func removeDockerContainer(name string) {
	_ = exec.Command("docker", "rm", "--force", name).Run()
}
