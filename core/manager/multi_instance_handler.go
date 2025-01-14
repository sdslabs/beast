package manager

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"os/exec"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"
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

// Rsync any file to other servers for chall deployment
func RsyncFileToServer(server config.AvailableServer, localFilePath, remoteFilePath string) error {
	err := utils.ValidateDirExists(localFilePath)
	if err != nil {
		return fmt.Errorf("file %s does not exist: %s", localFilePath, err)
	}
	fmt.Printf("Rsyncing %s to %s:%s\n", localFilePath, server.Host, remoteFilePath)
	cmd := exec.Command("rsync", "-avz",
		"-e", fmt.Sprintf("ssh -i %s", server.SSHKeyPath),
		localFilePath,
		fmt.Sprintf("%s@%s:%s", server.Username, server.Host, remoteFilePath))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	fmt.Println("Result: " + out.String())
	return nil
}

// Pings the server to check if it is reachable.
func PingServer(server config.AvailableServer) {
	client, err := CreateSSHClient(server)
	if err != nil {
		log.Errorf("SSH connection to %s failed: %s\n", server.Host, err)
		return
	}
	defer client.Close()
	log.Printf("SSH ping to %s succeeded!\n", server.Host)
}

// Run the command passed as argument on the remote server
func RunCommandOnServer(server config.AvailableServer, cmd string) (string, error) {
	client, _ := CreateSSHClient(server)
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %s", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %s\nOutput: %s", err, output)
	}

	log.Debugf("Command output for cmd %s : %s\n", cmd, output)
	return string(output), nil
}

// Creates an SSH client to connect to the remote server.
func CreateSSHClient(remoteServer config.AvailableServer) (*ssh.Client, error) {

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

func ValidateFileRemoteExists(server config.AvailableServer, stagedChallengePath string) bool {
	output, err := RunCommandOnServer(server, fmt.Sprintf("test -e %s&&echo exists||echo not exists", stagedChallengePath))
	if err != nil {
		log.Errorf("Error while checking file existence: %s\n", err)
		return false
	}
	log.Printf("Output: %s\n", output)
	if strings.TrimSpace(output) == "exists" {
		return true
	} else {
		return false
	}
}

func StageChallRemote(server config.AvailableServer, challenge database.Challenge) error {
	client, err := CreateSSHClient(server)
	if err != nil {
		return fmt.Errorf("SSH connection to %s failed: %s", server.Host, err)
	}
	defer client.Close()

	stagingDirPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	stagingRemoteDirPath := filepath.Join("~/.beast", core.BEAST_STAGING_DIR)
	// err = RunCommandOnServer(server, fmt.Sprintf("mkdir -p %s/%s", remoteStagingDir, challenge.Name))
	// if err != nil {
	// 	return fmt.Errorf("failed to create directory: %s", err)
	// }

	// Rsync the challenge files to the server
	err = RsyncFileToServer(server, fmt.Sprintf("%s/%s", stagingDirPath, challenge.Name), stagingRemoteDirPath)
	if err != nil {
		return fmt.Errorf("failed to rsync challenge files: %s", err)
	}

	return nil
}

// BuildImageFromTarContextRemote builds a Docker image from the tar context on the remote server.
//
//	TODO: Sidecar's configuration left. Need to be added.
func BuildImageFromTarContextRemote(challengeName string, imageTag string, stagedDir string, server config.AvailableServer) ([]byte, string, error) {
	remoteExtractPath := filepath.Join("~/.beast", core.BEAST_STAGING_DIR, challengeName, challengeName)
	_, err := RunCommandOnServer(server, fmt.Sprintf("mkdir -p %s && tar -xf %s -C %s", remoteExtractPath, stagedDir, remoteExtractPath))
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to extract tar: %s", err)
	}
	dockerBuildCmd := fmt.Sprintf("cd %s && docker build -t %s .", remoteExtractPath, imageTag)
	output, err := RunCommandOnServer(server, dockerBuildCmd)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to build docker image: %s\nOutput: %s", err, output)
	}
	getImageIDCmd := fmt.Sprintf(
		"docker images --format '{{.Repository}} {{.ID}}' | grep %s | awk '{print $2}'",
		imageTag,
	)
	imageID, err := RunCommandOnServer(server, getImageIDCmd)
	if err != nil {
		log.Fatalf("Failed to retrieve Docker image ID: %v", err)
	}
	if imageID == "" {
		return []byte{}, "", fmt.Errorf("failed to retrieve Docker image ID")
	}
	return []byte(output), strings.TrimSpace(imageID), nil
}

func CreateContainerFromImageRemote(containerConfig cr.CreateContainerConfig, server config.AvailableServer) (string, error) {
	var containerName, containerEnv, exposedPorts, portMap, cpuLimit, memoryLimit, pidLimit, imageID, mountBindings string
	if containerConfig.ContainerName != "" {
		containerName = fmt.Sprintf("--name %s ", containerConfig.ContainerName)
	}
	for _, envVar := range containerConfig.ContainerEnv {
		containerEnv += fmt.Sprintf("--env %s ", envVar)
	}
	for _, portMapping := range containerConfig.PortMapping {
		portMap += fmt.Sprintf("-p 0.0.0.0:%d:%d/%s ", portMapping.ContainerPort, portMapping.HostPort, containerConfig.TrafficType())
		exposedPorts += fmt.Sprintf("--expose %d ", portMapping.ContainerPort)
	}
	if containerConfig.CPUShares != 0 {
		cpuLimit = fmt.Sprintf("--cpu-shares %d ", containerConfig.CPUShares)
	}
	if containerConfig.Memory != 0 {
		memoryLimit = fmt.Sprintf("--memory %d ", containerConfig.Memory)
	}
	if containerConfig.PidsLimit != 0 {
		pidLimit = fmt.Sprintf("--pids-limit %d ", containerConfig.PidsLimit)
	}
	if containerConfig.ImageId != "" {
		imageID = containerConfig.ImageId
	}
	for src, dest := range containerConfig.MountsMap {
		mountBindings += fmt.Sprintf("--mount type=bind,source=%s,target=%s ", src, dest)
	}
	dockerCommand := fmt.Sprintf("docker run -d %s %s %s %s %s %s %s %s %s", containerName, containerEnv, exposedPorts, mountBindings, cpuLimit, memoryLimit, pidLimit, portMap, imageID)
	fmt.Println(dockerCommand)
	// fmt.Printf("%s, %s, %s, %s\n", containerName, containerEnv, exposedPorts, portMap)
	// dockerCommand := fmt.Sprintf("docker run \\
	// 	--name <container_name> \\
	// 	--env KEY1=value1 --env KEY2=value2 \\
	// 	--expose <internal_port> \\
	// 	--mount type=bind,source=<host_path>,target=<container_path> \\
	// 	--cpus=<cpu_limit> \\
	// 	--memory=<memory_limit> \\
	// 	--pids-limit <pid_limit> \\
	// 	-p 0.0.0.0:<external_port>:<internal_port> \\
	// 	<image_id>"
	// );
	output, err := RunCommandOnServer(server, dockerCommand)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %s\nOutput: %s", err, output)
	}
	log.Println(output[:12])
	return strings.TrimSpace(output[:12]), nil

	log.Println(containerConfig)
	return "", nil
}
