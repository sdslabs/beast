package remoteManager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/pkg/cr"
	"golang.org/x/crypto/ssh"
)

type cappedCommandBuffer struct {
	buffer    bytes.Buffer
	remaining int
	truncated bool
	mutex     sync.Mutex
}

func (buffer *cappedCommandBuffer) Write(data []byte) (int, error) {
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

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func shellJoin(arguments []string) string {
	quoted := make([]string, len(arguments))
	for index, argument := range arguments {
		quoted[index] = shellQuote(argument)
	}
	return strings.Join(quoted, " ")
}

func ExecContainerRemote(ctx context.Context, server config.AvailableServer, containerID string, command []string, outputLimit int) (cr.ExecResult, error) {
	if containerID == "" || len(command) == 0 {
		return cr.ExecResult{}, fmt.Errorf("container ID and command are required")
	}
	if outputLimit <= 0 {
		outputLimit = cr.DefaultExecOutputLimit
	}
	client, err := CreateSSHClient(server)
	if err != nil {
		return cr.ExecResult{}, err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return cr.ExecResult{}, fmt.Errorf("create SSH session: %w", err)
	}
	defer session.Close()

	stdout := &cappedCommandBuffer{remaining: (outputLimit + 1) / 2}
	stderr := &cappedCommandBuffer{remaining: outputLimit / 2}
	session.Stdout = stdout
	session.Stderr = stderr
	arguments := append([]string{"docker", "exec", "--user", "0", containerID}, command...)
	if err := session.Start(shellJoin(arguments)); err != nil {
		return cr.ExecResult{}, fmt.Errorf("start remote container exec: %w", err)
	}
	waitResult := make(chan error, 1)
	go func() { waitResult <- session.Wait() }()

	select {
	case <-ctx.Done():
		_ = session.Close()
		_ = client.Close()
		<-waitResult
		return cr.ExecResult{}, ctx.Err()
	case err := <-waitResult:
		exitCode := 0
		if err != nil {
			var exitError *ssh.ExitError
			if !errors.As(err, &exitError) {
				return cr.ExecResult{}, fmt.Errorf("execute remote container command: %w", err)
			}
			exitCode = exitError.ExitStatus()
		}
		return cr.ExecResult{
			Stdout:    stdout.buffer.String(),
			Stderr:    stderr.buffer.String(),
			ExitCode:  exitCode,
			Truncated: stdout.truncated || stderr.truncated,
		}, nil
	}
}
