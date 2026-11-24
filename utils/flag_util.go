package utils

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GetRandomFlagFromTemplate generates a concrete flag string from a template and dynamic inputs.
func GetRandomFlagFromTemplate(template string) (string, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return "", errors.New("flag template cannot be empty")
	}
	return template, nil
}

// RunGeneratorWithTimeout executes the challenge asset/flag generator script at generatorPath with a specified timeout.
// It passes the flag and the absolute path to an output directory as command-line arguments:
//
//	<generatorPath> <flag> <outputDirPath>
func RunGeneratorWithTimeout(generatorPath, flag, outputDirPath string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, generatorPath, flag, outputDirPath)
	cmd.Dir = filepath.Dir(generatorPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("generator timed out after %v: %w", timeout, err)
		}
		if len(output) > 0 {
			return fmt.Errorf("generator execution failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
		}
		return fmt.Errorf("generator execution failed: %w", err)
	}
	return nil
}

// RunGenerator executes the challenge asset/flag generator script at generatorPath with default 30s timeout.
func RunGenerator(generatorPath, flag, outputDirPath string) error {
	return RunGeneratorWithTimeout(generatorPath, flag, outputDirPath, 30*time.Second)
}
