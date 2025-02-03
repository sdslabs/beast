package remoteManager

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

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
}

// Stops and remove cremote container.
// Takes containerID and server config if both is available
// else just take containerID and find the server config from db
func StopAndRemoveContainerRemote(containerId string, server config.AvailableServer) error {
	if server == (config.AvailableServer{}) {
		chall, err := database.QueryChallengeEntries("id", containerId)
		if err != nil {
			if err == (gorm.ErrRecordNotFound) {
				log.Debugf("no container with container id %s present", containerId)
				return nil
			}
			return fmt.Errorf("DATABASE ERROR while fetching user details.")
		}
		if len(chall) > 0 {
			server = config.Cfg.AvailableServers[chall[0].ServerDeployed]
		} else {
			return fmt.Errorf("no container with container id %s found", containerId)
		}
	}
	stopCommand := fmt.Sprintf("docker stop %s", containerId)
	if _, err := RunCommandOnServer(server, stopCommand); err != nil {
		return fmt.Errorf("failed to stop container on server %s : %w", server.Host, err)
	}
	log.Debugf("Stopped container with ID %s on %s", containerId, server.Host)

	removeCommand := fmt.Sprintf("docker rm --force %s", containerId)
	if _, err := RunCommandOnServer(server, removeCommand); err != nil {
		return fmt.Errorf("failed to remove container on server %s : %w", server.Host, err)
	}
	log.Printf("Removed container with ID %s on %s", containerId, server.Host)

	return nil
}

// Function searches containers based on the filter map on all remote servers
func SearchContainerByFilterRemote(filterMap map[string]string, server config.AvailableServer) ([]types.Container, error) {
	filterArgs := ""
	containers := []types.Container{}
	var output string
	var err error
	for key, val := range filterMap {
		filterArgs += fmt.Sprintf("--filter='%s=%s' ", key, val)
	}
	if server == (config.AvailableServer{}) {
		for _, server := range config.Cfg.AvailableServers {
			if server.Active {
				if server.Host != "localhost" {
					output, err = RunCommandOnServer(server, fmt.Sprintf("docker ps -a %s --format '{{.ID}}'", filterArgs))
					if err != nil {
						return []types.Container{}, err
					}
					for _, line := range bytes.Split([]byte(output), []byte("\n")) {
						if len(line) > 0 {
							containers = append(containers, types.Container{ID: string(line)})
						}
					}
				}
			}
		}
	} else {
		if server.Active {
			if server.Host != "localhost" {
				output, err = RunCommandOnServer(server, fmt.Sprintf("docker ps -a %s --format '{{.ID}}'", filterArgs))
				if err != nil {
					return []types.Container{}, err
				}
				for _, line := range bytes.Split([]byte(output), []byte("\n")) {
					if len(line) > 0 {
						containers = append(containers, types.Container{ID: string(line)})
					}
				}
			}
		}
	}

	return containers, nil
}

// Function searches for running containers based on the filter map on all remote server
func SearchRunningContainerByFilterRemote(filterMap map[string]string, server config.AvailableServer) ([]types.Container, error) {
	filterArgs := ""
	containers := []types.Container{}
	var output string
	var err error
	for key, val := range filterMap {
		filterArgs += fmt.Sprintf("--filter='%s=%s' ", key, val)
	}
	if server == (config.AvailableServer{}) {
		for _, server := range config.Cfg.AvailableServers {
			if server.Active {
				if server.Host != "localhost" {
					output, err = RunCommandOnServer(server, fmt.Sprintf("docker ps %s --format '{{.ID}}'", filterArgs))
					if err != nil {
						return []types.Container{}, err
					}
					for _, line := range bytes.Split([]byte(output), []byte("\n")) {
						if len(line) > 0 {
							containers = append(containers, types.Container{ID: string(line)})
						}
					}
				}
			}
		}
	} else {
		if server.Active {
			if server.Host != "localhost" {
				output, err = RunCommandOnServer(server, fmt.Sprintf("docker ps -a %s --format '{{.ID}}'", filterArgs))
				if err != nil {
					return []types.Container{}, err
				}
				for _, line := range bytes.Split([]byte(output), []byte("\n")) {
					if len(line) > 0 {
						containers = append(containers, types.Container{ID: string(line)})
					}
				}
			}
		}
	}

	return containers, nil
}

// Get Containers stdout, stderr logs
func GetContainerStdLogsRemote(containerID string, server config.AvailableServer) (*cr.Log, error) {
	stdoutCmd := fmt.Sprintf("docker logs --details --stdout %s", containerID)
	stderrCmd := fmt.Sprintf("docker logs --details --stderr %s", containerID)

	stdout, err := RunCommandOnServer(server, stdoutCmd)
	if err != nil {
		return nil, fmt.Errorf("error fetching stdout logs: %w", err)
	}

	stderr, err := RunCommandOnServer(server, stderrCmd)
	if err != nil {
		return nil, fmt.Errorf("error fetching stderr logs: %w", err)
	}

	return &cr.Log{Stdout: stdout, Stderr: stderr}, nil
}

// Get live logs of container
func ShowLiveContainerLogsRemote(containerID string, server config.AvailableServer) error {
	command := fmt.Sprintf("docker logs --details --follow %s", containerID)

	output, err := RunCommandOnServer(server, command)
	if err != nil {
		return fmt.Errorf("error streaming live logs: %w", err)
	}

	fmt.Println(output)
	return nil
}

// Commit container on remote server
func CommitContainerRemote(containerID string, server config.AvailableServer) (string, error) {
	command := fmt.Sprintf("docker commit %s", containerID)

	output, err := RunCommandOnServer(server, command)
	if err != nil {
		return "", fmt.Errorf("error committing container: %w", err)
	}
	imageID := strings.TrimSpace(output)
	return imageID, nil
}
