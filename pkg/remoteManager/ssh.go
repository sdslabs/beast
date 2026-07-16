package remoteManager

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type LoadBalancerQueue struct {
	servers []config.AvailableServer
	mu      sync.Mutex
}

var ServerQueue LoadBalancerQueue

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type RemoteCommandError struct {
	ExitStatus int
	Output     string
	Err        error
}

func (commandError *RemoteCommandError) Error() string {
	return fmt.Sprintf("remote command exited with status %d: %v", commandError.ExitStatus, commandError.Err)
}

func (commandError *RemoteCommandError) Unwrap() error {
	return commandError.Err
}

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
		exitStatus := -1
		var exitError *ssh.ExitError
		if errors.As(err, &exitError) {
			exitStatus = exitError.ExitStatus()
		}
		return string(output), &RemoteCommandError{ExitStatus: exitStatus, Output: string(output), Err: err}
	}

	log.Debugf("Command output for cmd %s : %s\n", cmd, output)
	return string(output), nil
}

func RunArgsOnServer(server config.AvailableServer, arguments ...string) (string, error) {
	if len(arguments) == 0 {
		return "", fmt.Errorf("remote command arguments are empty")
	}
	return RunCommandOnServer(server, "exec "+shellJoin(arguments))
}

func RunArgsInDirOnServer(server config.AvailableServer, directory string, arguments ...string) (string, error) {
	if directory == "" || len(arguments) == 0 {
		return "", fmt.Errorf("remote directory and command arguments are required")
	}
	command := "cd -- " + shellQuote(directory) + " && exec " + shellJoin(arguments)
	return RunCommandOnServer(server, command)
}

func RunArgsWithEnvOnServer(server config.AvailableServer, environment map[string]string, arguments ...string) (string, error) {
	if len(arguments) == 0 {
		return "", fmt.Errorf("remote command arguments are empty")
	}
	keys := make([]string, 0, len(environment))
	for key := range environment {
		if !environmentNamePattern.MatchString(key) {
			return "", fmt.Errorf("invalid environment variable name %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	assignments := make([]string, 0, len(keys))
	for _, key := range keys {
		assignments = append(assignments, key+"="+shellQuote(environment[key]))
	}
	command := strings.Join(assignments, " ")
	if command != "" {
		command += " "
	}
	command += "exec " + shellJoin(arguments)
	return RunCommandOnServer(server, command)
}

// Creates an SSH client to connect to the remote server.
func CreateSSHClient(remoteServer config.AvailableServer) (*ssh.Client, error) {
	if !remoteServer.Active {
		return nil, fmt.Errorf("server is inactive in config.toml")
	}
	hostKeyCallback, err := knownhosts.New(remoteServer.KnownHostsFile)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts file: %s", err)
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
		HostKeyCallback: hostKeyCallback,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", remoteServer.Host), config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %s", err)
	}
	return client, nil
}
