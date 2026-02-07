package remoteManager

import (
	"errors"
	"fmt"
	"github.com/sdslabs/beastv4/pkg/cr"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type LoadBalancerQueue struct {
	servers []config.AvailableServer
	mu      sync.Mutex
}

var ServerQueue LoadBalancerQueue

// Returns a Queue of all available server to achive Round-Robin load balancing
func NewLoadBalancerQueue() LoadBalancerQueue {
	return LoadBalancerQueue{}
}

// Push adds a server to the queue.
func (q *LoadBalancerQueue) Push(server config.AvailableServer) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.servers = append(q.servers, server)
}

// Pop removes the server from top of queue.
func (q *LoadBalancerQueue) Pop() (config.AvailableServer, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.servers) == 0 {
		return config.AvailableServer{}, errors.New("queue is empty")
	}

	server := q.servers[0]
	q.servers = q.servers[1:]
	return server, nil
}

// GetNextAvailableInstance returns the next available server for load balancing.
func (q *LoadBalancerQueue) GetNextAvailableInstance() (config.AvailableServer, error) {
	avail_server, err := q.Pop()
	if err != nil {
		return config.AvailableServer{}, err
	}
	q.Push(avail_server)
	return avail_server, nil
}

// Pings the server to check if it is reachable.
func PingServer(server config.AvailableServer) error {
	if !server.Active {
		return fmt.Errorf("server is inactive in config.toml")
	}
	client, err := CreateSSHClient(server)
	if err != nil {
		err = fmt.Errorf("SSH connection to %s failed: %s\n", server.Host, err)
		log.Error(err)
		return err
	}
	defer client.Close()
	log.Printf("SSH ping to %s succeeded!\n", server.Host)
	return nil
}

// Run the command passed as argument on the remote server
func RunCommandOnServer(server config.AvailableServer, cmd string) (string, error) {
	if !server.Active {
		return "", fmt.Errorf("server is inactive in config.toml")
	}
	client, err := CreateSSHClient(server)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %s", err)
	}
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %s", err)
	}
	defer client.Close()
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %s\nOutput: %s", err, output)
	}

	log.Debugf("Command output for cmd %s : %s\n", cmd, output)
	return string(output), nil
}

// Runs a command in a container on a remote server
func RunCommandInContainerOnServer(server config.AvailableServer, containerId string, cmd string) (cr.ExecResult, error) {
	result := cr.ExecResult{
		ExitCode: 1,
	}

	if !server.Active {
		return result, fmt.Errorf("server is inactive in config.toml")
	}
	client, err := CreateSSHClient(server)
	if err != nil {
		return result, fmt.Errorf("failed to create session: %s", err)
	}
	session, err := client.NewSession()
	if err != nil {
		return result, fmt.Errorf("failed to create session: %s", err)
	}
	defer session.Close()

	/* Run the command, capture the output and exit code and emit them in a specific order */
	dockerCmd := fmt.Sprintf(`
		output=$(docker exec %s sh -c "%s"); 
		exitcode=$?;
		echo "$exitcode"
		echo "$output"
	`, containerId, cmd)

	output, err := session.CombinedOutput(dockerCmd)
	if err != nil {
		return result, fmt.Errorf("failed to execute command: %s\nOutput: %s", err, output)
	}

	lines := strings.SplitN(string(output), "\n", 2)
	result.ExitCode, err = strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return result, fmt.Errorf("failed to parse exit code: %s\nOutput: %s", err, output)
	}

	if len(lines) > 1 {
		result.Output = lines[1]
	}

	log.Debugf("Command output for cmd %s : %s\n", cmd, result.Output)
	log.Debugf("Exit Code for cmd %s : %s\n", cmd, result.ExitCode)
	return result, nil
}

// Creates an SSH client to connect to the remote server.
func CreateSSHClient(remoteServer config.AvailableServer) (*ssh.Client, error) {
	if !remoteServer.Active {
		return nil, fmt.Errorf("server is inactive in config.toml")
	}
	key, err := ioutil.ReadFile(remoteServer.SSHKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %s", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %s", err)
	}

	config := &ssh.ClientConfig{
		User: remoteServer.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Insecure for now. Integrate proper callback for host key verification.
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", remoteServer.Host), config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %s", err)
	}
	return client, nil
}
