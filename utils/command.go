package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

const maxCommandOutput = 64 << 10

var ErrCommandOutputLimit = errors.New("command output exceeds 64 KiB limit")

type boundedCommandOutput struct {
	mutex     sync.Mutex
	buffer    bytes.Buffer
	remaining int
	truncated bool
}

func (output *boundedCommandOutput) Write(data []byte) (int, error) {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	written := len(data)
	if len(data) > output.remaining {
		data = data[:output.remaining]
		output.truncated = true
	}
	if len(data) > 0 {
		_, _ = output.buffer.Write(data)
		output.remaining -= len(data)
	}
	return written, nil
}

func (output *boundedCommandOutput) String() string {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	return output.buffer.String()
}

func RunCommand(timeout time.Duration, environment []string, name string, arguments ...string) (string, error) {
	if timeout <= 0 {
		return "", fmt.Errorf("command timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	if environment != nil {
		command.Env = environment
	}
	output := &boundedCommandOutput{remaining: maxCommandOutput}
	command.Stdout = output
	command.Stderr = output
	err := command.Run()
	if ctx.Err() != nil {
		return output.String(), fmt.Errorf("command timed out after %s: %w", timeout, ctx.Err())
	}
	if output.truncated {
		return output.String(), ErrCommandOutputLimit
	}
	return output.String(), err
}
