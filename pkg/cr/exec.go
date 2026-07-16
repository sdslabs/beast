package cr

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
)

const DefaultExecOutputLimit = 1 << 20

type ExecResult struct {
	Stdout    string
	Stderr    string
	ExitCode  int
	Truncated bool
}

type cappedBuffer struct {
	buffer    bytes.Buffer
	remaining int
	truncated bool
	mutex     sync.Mutex
}

func (buffer *cappedBuffer) Write(data []byte) (int, error) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	written := len(data)
	if len(data) > buffer.remaining {
		data = data[:buffer.remaining]
		buffer.truncated = true
	}
	if len(data) > 0 {
		_, _ = buffer.buffer.Write(data)
		buffer.remaining -= len(data)
	}
	return written, nil
}

func (buffer *cappedBuffer) String() string {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return buffer.buffer.String()
}

func (buffer *cappedBuffer) Truncated() bool {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	return buffer.truncated
}

func ExecContainer(ctx context.Context, containerID string, command []string, outputLimit int) (ExecResult, error) {
	if containerID == "" {
		return ExecResult{}, fmt.Errorf("container ID is empty")
	}
	if len(command) == 0 {
		return ExecResult{}, fmt.Errorf("command is empty")
	}
	if outputLimit <= 0 {
		outputLimit = DefaultExecOutputLimit
	}

	client, err := newDockerClient()
	if err != nil {
		return ExecResult{}, fmt.Errorf("create Docker client: %w", err)
	}
	exec, err := client.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		User:         "0",
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          command,
	})
	if err != nil {
		return ExecResult{}, fmt.Errorf("create container exec: %w", err)
	}
	response, err := client.ContainerExecAttach(ctx, exec.ID, types.ExecStartCheck{})
	if err != nil {
		return ExecResult{}, fmt.Errorf("attach container exec: %w", err)
	}
	defer response.Close()

	stopCancellation := make(chan struct{})
	defer close(stopCancellation)
	go func() {
		select {
		case <-ctx.Done():
			response.Close()
		case <-stopCancellation:
		}
	}()

	stdout := &cappedBuffer{remaining: (outputLimit + 1) / 2}
	stderr := &cappedBuffer{remaining: outputLimit / 2}
	if _, err := stdcopy.StdCopy(stdout, stderr, response.Reader); err != nil && ctx.Err() == nil {
		return ExecResult{}, fmt.Errorf("read container exec output: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return ExecResult{}, err
	}
	inspection, err := client.ContainerExecInspect(ctx, exec.ID)
	if err != nil {
		return ExecResult{}, fmt.Errorf("inspect container exec: %w", err)
	}

	return ExecResult{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		ExitCode:  inspection.ExitCode,
		Truncated: stdout.Truncated() || stderr.Truncated(),
	}, nil
}
