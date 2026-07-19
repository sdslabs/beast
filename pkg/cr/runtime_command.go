package cr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

const (
	runtimeCommandTimeout   = 2 * time.Minute
	maxRuntimeCommandOutput = 1 << 20
)

var errRuntimeCommandOutputLimit = errors.New("runtime command output exceeds 1 MiB limit")

type runtimeCommandOptions struct {
	directory   string
	environment []string
	timeout     time.Duration
}

type boundedRuntimeOutput struct {
	mutex     sync.Mutex
	buffer    bytes.Buffer
	remaining int
	truncated bool
}

func (output *boundedRuntimeOutput) Write(data []byte) (int, error) {
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

func (output *boundedRuntimeOutput) String() string {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	return output.buffer.String()
}

func runRuntimeCommand(name string, arguments []string, options runtimeCommandOptions) (string, error) {
	timeout := options.timeout
	if timeout <= 0 {
		timeout = runtimeCommandTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = options.directory
	if options.environment != nil {
		command.Env = options.environment
	}
	output := &boundedRuntimeOutput{remaining: maxRuntimeCommandOutput}
	command.Stdout = output
	command.Stderr = output

	err := command.Run()
	outputText := output.String()
	if ctx.Err() != nil {
		return outputText, fmt.Errorf("%s timed out after %s: %w", name, timeout, ctx.Err())
	}
	if output.truncated {
		return outputText, errRuntimeCommandOutputLimit
	}
	return outputText, err
}
