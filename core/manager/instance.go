package manager

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

func SpawnInstance(challengeName, userID, username string, userSSHKey string) (*cache.Instance, error) {
	log.Infof("Spawning instance of challenge %s for user %s", challengeName, userID)

	existingInstance, err := cache.GetUserInstance(userID, challengeName)
	if err == nil && existingInstance != nil {
		return existingInstance, fmt.Errorf("user already has an active instance of this challenge")
	}

	instanceCount, err := cache.CountUserInstances(userID)
	if err != nil {
		log.Warnf("Failed to count user instances: %v", err)
	} else if instanceCount >= cfg.Cfg.InstanceConfig.MaxInstancesPerUser {
		return nil, fmt.Errorf("maximum instances limit reached (%d)", cfg.Cfg.InstanceConfig.MaxInstancesPerUser)
	}

	challenge, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		return nil, fmt.Errorf("failed to query challenge: %w", err)
	}

	challengeStagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	configFile := filepath.Join(challengeStagingDir, core.CHALLENGE_CONFIG_FILE_NAME)

	var config cfg.BeastChallengeConfig
	_, err = toml.DecodeFile(configFile, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to load challenge config: %w", err)
	}

	config.Resources.ValidateRequiredFields()

	if !config.Challenge.Metadata.IsInstanced() {
		return nil, fmt.Errorf("challenge %s is not configured for instancing", challengeName)
	}

	if challenge.ImageId == "" && config.Challenge.Env.DockerCompose == "" {
		return nil, fmt.Errorf("challenge %s has not been committed (no image available)", challengeName)
	}

	serverDeployed := selectServerForInstance()

	instanceID := uuid.New().String()[:12]

	expirationSeconds := config.Challenge.Metadata.GetInstanceExpiration()
	ttl := time.Duration(expirationSeconds) * time.Second
	expiresAt := time.Now().Add(ttl)

	var port uint32
	var checkHash string
	var containerID string
	var portOwner string
	var deploymentType string

	if config.Challenge.Env.DockerCompose != "" {
		err = config.Challenge.Env.ExtractPortsCompose(challengeStagingDir)
		if err != nil {
			return nil, fmt.Errorf("failed to extract port variables from compose file: %s", err.Error())
		}

		ports, err := allocateInstancePortsCompose(serverDeployed, config.Challenge.Env)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate instance ports: %s", err.Error())
		}

		port = ports[config.Challenge.Env.DefaultPortVar]

		containerID, checkHash, err = deployInstanceFromCompose(instanceID, challengeName, &config, challengeStagingDir, serverDeployed, ports)
		portOwner = utils.ComposeDockerProjectNameInstanced(challengeName, instanceID)
		deploymentType = core.DEPLOYMENT_TYPES["docker_compose"]

		if err != nil {
			coreUtils.FreePortsOnHostCompose(serverDeployed, ports)
			return nil, err
		}

		if serverDeployed == core.LOCALHOST || serverDeployed == "" {
			err = addUserSSHKeyLocal(containerID, userSSHKey)
		} else {
			err = addUserSSHKeyRemote(containerID, cfg.Cfg.AvailableServers[serverDeployed], userSSHKey)
		}

		if err != nil {
			if cleanupErr := killInstanceContainer(containerID, deploymentType, instanceID, challengeName, serverDeployed); cleanupErr != nil {
				log.Warnf("failed to cleanup instance %s after ssh key injection failure: %v", instanceID, cleanupErr)
			}
			coreUtils.FreePortsOnHostCompose(serverDeployed, ports)
			return nil, fmt.Errorf("failed to add user ssh key: %s", err.Error())
		}

		if err := coreUtils.AssignPortsOnContainerToHostCompose(serverDeployed, portOwner, ports); err != nil {
			if cleanupErr := killInstanceContainer(containerID, deploymentType, instanceID, challengeName, serverDeployed); cleanupErr != nil {
				log.Warnf("failed to cleanup instance %s after port registration failure: %v", instanceID, cleanupErr)
			}
			coreUtils.FreePortsOnHostCompose(serverDeployed, ports)
			return nil, fmt.Errorf("failed to register instance ports: %w", err)
		}
	} else {
		err = config.Challenge.Env.ExtractPorts()
		if err != nil {
			return nil, fmt.Errorf("failed to extract port variables from compose file: %s", err.Error())
		}

		ports, err := allocateInstancePorts(serverDeployed, config.Challenge.Env)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate instance ports: %s", err.Error())
		}

		var found bool
		found, port = utils.Uint32InIndexList(config.Challenge.Env.DefaultPort, config.Challenge.Env.Ports, ports)
		if !found {
			coreUtils.FreePortsOnHost(serverDeployed, ports)
			return nil, fmt.Errorf("failed to allocate instance port for challenge %s", challengeName)
		}

		containerID, checkHash, err = deployInstanceContainer(instanceID, challengeName, challenge.ImageId, &config, serverDeployed, ports)
		portOwner = containerID
		deploymentType = core.DEPLOYMENT_TYPES["standard_docker"]

		if err != nil {
			coreUtils.FreePortsOnHost(serverDeployed, ports)
			return nil, fmt.Errorf("error while creating container for challenge %s: %s", challenge.Name, err.Error())
		}

		if serverDeployed == core.LOCALHOST || serverDeployed == "" {
			err = addUserSSHKeyLocal(containerID, userSSHKey)
		} else {
			err = addUserSSHKeyRemote(containerID, cfg.Cfg.AvailableServers[serverDeployed], userSSHKey)
		}

		if err != nil {
			if cleanupErr := killInstanceContainer(containerID, deploymentType, instanceID, challengeName, serverDeployed); cleanupErr != nil {
				log.Warnf("failed to cleanup instance %s after ssh key injection failure: %v", instanceID, cleanupErr)
			}
			coreUtils.FreePortsOnHost(serverDeployed, ports)
			return nil, fmt.Errorf("failed to add user ssh key: %s", err.Error())
		}

		if err := coreUtils.AssignPortsOnContainerToHost(serverDeployed, containerID, ports); err != nil {
			if cleanupErr := killInstanceContainer(containerID, deploymentType, instanceID, challengeName, serverDeployed); cleanupErr != nil {
				log.Warnf("failed to cleanup instance %s after port registration failure: %v", instanceID, cleanupErr)
			}
			coreUtils.FreePortsOnHost(serverDeployed, ports)
			return nil, fmt.Errorf("failed to register instance ports: %w", err)
		}
	}

	instance := &cache.Instance{
		InstanceID:     instanceID,
		ChallengeName:  challengeName,
		ContainerID:    containerID,
		PortOwner:      portOwner,
		CheckHash:      checkHash,
		Port:           port,
		UserID:         userID,
		Username:       username,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
		DeploymentType: deploymentType,
		ServerDeployed: serverDeployed,
	}

	err = cache.SaveInstance(instance, ttl)
	if err != nil {
		if err := killInstanceContainer(containerID, deploymentType, instanceID, challengeName, serverDeployed); err != nil {
			return nil, fmt.Errorf("failed to kill instance container: %w", err)
		}
		if err := cache.FreeContainerPortsOnHost(serverDeployed, portOwner); err != nil {
			return nil, fmt.Errorf("failed to free container ports: %w", err)
		}

		return nil, fmt.Errorf("failed to save instance: %w", err)
	}

	log.Infof("Successfully spawned instance %s for user %s, challenge %s on port %d (server: %s)",
		instanceID, userID, challengeName, port, serverDeployed)

	return instance, nil
}

func KillInstance(instanceID string) error {
	log.Infof("Killing instance %s", instanceID)

	instance, err := cache.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	err = killInstanceContainer(instance.ContainerID, instance.DeploymentType, instanceID, instance.ChallengeName, instance.ServerDeployed)
	if err != nil {
		log.Warnf("Error killing container for instance %s: %v", instanceID, err)
	}

	err = cache.FreeContainerPortsOnHost(instance.ServerDeployed, instance.PortOwnerID())
	if err != nil {
		return fmt.Errorf("failed to free container ports: %w", err)
	}

	err = cache.DeleteInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to delete instance from cache: %w", err)
	}

	log.Infof("Successfully killed instance %s", instanceID)
	return nil
}

// KillUserInstance kills a user's instance of a specific challenge
func KillUserInstance(userID, challengeName string) error {
	instance, err := cache.GetUserInstance(userID, challengeName)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	return KillInstance(instance.InstanceID)
}

// ExtendInstance extends the lifetime of an instance
func ExtendInstance(instanceID string, additionalSeconds int64) error {
	// Check max extension limit
	maxExtension := cfg.Cfg.InstanceConfig.MaxExtension
	if additionalSeconds > maxExtension {
		additionalSeconds = maxExtension
	}

	additionalTime := time.Duration(additionalSeconds) * time.Second
	return cache.ExtendInstance(instanceID, additionalTime)
}

// GetInstance retrieves an instance by ID
func GetInstance(instanceID string) (*cache.Instance, error) {
	return cache.GetInstance(instanceID)
}

// GetUserInstance retrieves a user's instance of a challenge
func GetUserInstance(userID, challengeName string) (*cache.Instance, error) {
	return cache.GetUserInstance(userID, challengeName)
}

// GetUserInstances retrieves all instances for a user
func GetUserInstances(userID string) ([]*cache.Instance, error) {
	return cache.GetUserInstances(userID)
}

// GetAllInstances retrieves all active instances (admin only)
func GetAllInstances() ([]*cache.Instance, error) {
	return cache.GetAllInstances()
}

// GetChallengeInstances retrieves all active instances for a specific challenge
func GetChallengeInstances(challengeName string) ([]*cache.Instance, error) {
	return cache.GetChallengeInstances(challengeName)
}

// KillChallengeInstances kills all active instances of a challenge.
// This should be called when undeploying or purging a challenge.
func KillChallengeInstances(challengeName string) error {
	instances, err := cache.GetChallengeInstances(challengeName)
	if err != nil {
		return fmt.Errorf("failed to get instances for challenge %s: %w", challengeName, err)
	}

	if len(instances) == 0 {
		log.Debugf("No active instances found for challenge %s", challengeName)
		return nil
	}

	log.Infof("Killing %d active instance(s) for challenge %s", len(instances), challengeName)

	var lastErr error
	for _, instance := range instances {
		log.Infof("Killing instance %s for user %s (challenge: %s)",
			instance.InstanceID, instance.UserID, challengeName)

		if err := KillInstance(instance.InstanceID); err != nil {
			log.Warnf("Failed to kill instance %s: %v", instance.InstanceID, err)
			lastErr = err
		}
	}

	return lastErr
}

func allocateInstancePorts(host string, env cfg.ChallengeEnv) ([]uint32, error) {
	var firstPort, lastPort uint32
	var err error

	server := cfg.Cfg.AvailableServers[host]
	firstPort, lastPort, err = utils.ParsePortMapping(server.PortRange)

	if err != nil {
		return nil, fmt.Errorf("failed to parse port range: %w", err)
	}

	portRange := lastPort - firstPort + 1

	ports, err := cache.GetFreePortsOnHost(host, firstPort, portRange, len(env.Ports))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate ports: %w", err)
	}

	return ports, nil
}

func allocateInstancePortsCompose(host string, env cfg.ChallengeEnv) (map[string]uint32, error) {
	var err error
	var firstPort, lastPort uint32

	server := cfg.Cfg.AvailableServers[host]
	firstPort, lastPort, err = utils.ParsePortMapping(server.PortRange)

	if err != nil {
		return nil, fmt.Errorf("failed to parse port range: %w", err)
	}

	portRange := lastPort - firstPort + 1

	allocatedPorts, err := cache.GetFreePortsOnHost(host, firstPort, portRange, len(env.PortVariables))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate instance ports: %w", err)
	}

	ports := make(map[string]uint32, len(env.PortVariables))
	for i, portVariable := range env.PortVariables {
		ports[portVariable] = allocatedPorts[i]
	}

	return ports, nil
}

func selectServerForInstance() string {
	availableServer, err := remoteManager.ServerQueue.GetNextAvailableInstance()
	if err == nil && availableServer.Name != "" {
		return availableServer.Name
	}
	return core.LOCALHOST
}

func verifyCheckLocal(containerId string) (string, error) {
	fileCommand := fmt.Sprintf("[ -f '%s' ]", core.SAD_CHECK_SCRIPT_LOCATION)
	chmodCommand := fmt.Sprintf("command chmod +x %s", core.SAD_CHECK_SCRIPT_LOCATION)
	hashCommand := fmt.Sprintf("command cat %s | sha256sum", core.SAD_CHECK_SCRIPT_LOCATION)

	result, err := cr.RunCommandInContainer(containerId, []string{
		"sh", "-c", fileCommand,
	})

	if err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("failed to verify 'check.sh' at location: %s", core.SAD_CHECK_SCRIPT_LOCATION)
	}

	result, err = cr.RunCommandInContainer(containerId, []string{
		"sh", "-c", hashCommand,
	})

	if err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("failed to hash 'check.sh' to store flag")
	}

	hash := result.Output

	result, err = cr.RunCommandInContainer(containerId, []string{
		"sh", "-c", chmodCommand,
	})

	if err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("failed to make 'check.sh' executable at: %s", core.SAD_CHECK_SCRIPT_LOCATION)
	}

	return strings.TrimSpace(hash), nil
}

func verifyCheckRemote(containerId string, server cfg.AvailableServer) (string, error) {
	fileCommand := fmt.Sprintf("[ -f '%s' ]", core.SAD_CHECK_SCRIPT_LOCATION)
	chmodCommand := fmt.Sprintf("command chmod +x %s", core.SAD_CHECK_SCRIPT_LOCATION)
	hashCommand := fmt.Sprintf("command cat %s | sha256sum", core.SAD_CHECK_SCRIPT_LOCATION)

	result, err := remoteManager.RunCommandInContainerOnServer(server, containerId, fileCommand)

	if err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("failed to verify 'check.sh' at location: %s", core.SAD_CHECK_SCRIPT_LOCATION)
	}

	result, err = remoteManager.RunCommandInContainerOnServer(server, containerId, hashCommand)

	if err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("failed to hash 'check.sh' to store flag")
	}

	hash := result.Output

	result, err = remoteManager.RunCommandInContainerOnServer(server, containerId, chmodCommand)

	if err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("failed to make 'check.sh' executable at: %s", core.SAD_CHECK_SCRIPT_LOCATION)
	}

	return strings.TrimSpace(hash), nil
}

func verifySSHLocal(containerId string) error {
	err := exec.Command("docker", "port", containerId, fmt.Sprintf("%v/tcp", core.SSH_PORT)).Run()
	if err != nil {
		return err
	}

	sshAgentCheckCommand := fmt.Sprintf("[ -S \"$SSH_AUTH_SOCK\" ]")
	result, err := cr.RunCommandInContainer(containerId, []string{
		"sh", "-c", sshAgentCheckCommand,
	})

	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("ssh agent check failed, ssh-agent not running")
	}

	sshServiceStartCommand := fmt.Sprintf("service ssh start")
	result, err = cr.RunCommandInContainer(containerId, []string{
		"sh", "-c", sshServiceStartCommand,
	})

	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("ssh start failed, ssh agent not running")
	}

	return nil
}

func verifySSHRemote(containerId string, server cfg.AvailableServer) error {
	portCmd := fmt.Sprintf("docker port %s %v/tcp", containerId, core.SSH_PORT)
	_, err := remoteManager.RunCommandOnServer(server, portCmd)
	if err != nil {
		return err
	}

	sshAgentCheckCommand := fmt.Sprintf("[ -S \"$SSH_AUTH_SOCK\" ]")
	result, err := remoteManager.RunCommandInContainerOnServer(server, containerId, sshAgentCheckCommand)

	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("ssh agent check failed, ssh-agent not running")
	}

	sshServiceStartCommand := fmt.Sprintf("service ssh start")
	result, err = cr.RunCommandInContainer(containerId, []string{
		"sh", "-c", sshServiceStartCommand,
	})

	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("ssh start failed, ssh agent not running")
	}

	return err
}

func addUserSSHKeyLocal(containerId string, sshKey string) error {
	sshCmd := fmt.Sprintf("mkdir -p /home/beast/.ssh && echo '%s' >> /home/beast/.ssh/authorized_keys && chmod 700 /home/beast/.ssh && chmod 600 /home/beast/.ssh/authorized_keys && chown -R beast:beast-grp /home/beast/.ssh", sshKey)

	result, err := cr.RunCommandInContainer(containerId, []string{
		"sh", "-c", sshCmd,
	})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to add public key to container: %s", result.Output)
	}

	return nil
}

func addUserSSHKeyRemote(containerId string, server cfg.AvailableServer, sshKey string) error {
	sshCmd := fmt.Sprintf("mkdir -p /home/beast/.ssh && echo '%s' >> /home/beast/.ssh/authorized_keys && chmod 700 /home/beast/.ssh && chmod 600 /home/beast/.ssh/authorized_keys && chown -R beast:beast-grp /home/beast/.ssh", sshKey)

	result, err := remoteManager.RunCommandInContainerOnServer(server, containerId, sshCmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to add public key to container: %s", result.Output)
	}

	return nil
}

func deployInstanceContainer(instanceID, challengeName string, imageID string, config *cfg.BeastChallengeConfig, serverDeployed string, ports []uint32) (string, string, error) {
	// Instanced non compose challenges are managed by the container ID
	containerName := utils.ComposeDockerProjectNameInstanced(challengeName, instanceID)

	containerPort := config.Challenge.Env.DefaultPort
	if containerPort != core.SSH_PORT {
		return "", "", fmt.Errorf("the default port is not set to %d for instance challenge %s", core.SSH_PORT, challengeName)
	}

	portMapping := make([]cr.PortMapping, len(ports))
	for i, port := range ports {
		portMapping[i] = cr.PortMapping{
			HostPort:      port,
			ContainerPort: config.Challenge.Env.Ports[i],
		}
	}

	var containerEnv []string
	for _, env := range config.Challenge.Env.EnvironmentVars {
		containerEnv = append(containerEnv, fmt.Sprintf("%s=%s", env.Key, filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, env.Value)))
	}

	containerConfig := cr.CreateContainerConfig{
		PortMapping:   portMapping,
		MountsMap:     make(map[string]string),
		ImageId:       imageID,
		ContainerName: containerName,
		ChallengeName: challengeName,
		ContainerEnv:  containerEnv,
		Traffic:       config.Challenge.Env.TrafficType(),
		CPUsLimit:     config.Resources.CPUsLimit,
		CPUShares:     config.Resources.CPUShares,
		Memory:        config.Resources.Memory,
		PidsLimit:     config.Resources.PidsLimit,
		Labels: map[string]string{
			"beast.instance":    "true",
			"beast.instance.id": instanceID,
		},
	}

	var err error
	var checkHash string
	var containerId string

	if cfg.Cfg.UseLocalDockerDaemon(serverDeployed) {
		containerId, err = cr.CreateContainerFromImage(&containerConfig)
		if err != nil {
			return "", "", fmt.Errorf("failed to create container from image: %w", err)
		}

		err = verifySSHLocal(containerId)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify exposes of port %v in container %s on localhost", core.SSH_PORT, containerId)
		}

		checkHash, err = verifyCheckLocal(containerId)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify check.sh at location %s in container: %s on localhost: %w", core.SAD_CHECK_SCRIPT_LOCATION, containerId, err)
		}
	} else {
		server := cfg.Cfg.AvailableServers[serverDeployed]
		containerId, err = remoteManager.CreateContainerFromImageRemote(containerConfig, server)
		if err != nil {
			return "", "", fmt.Errorf("failed to create container from image: %w", err)
		}

		err = verifySSHRemote(containerId, server)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify exposes of port %v in container %s on host %s", core.SSH_PORT, containerId, server.Host)
		}

		checkHash, err = verifyCheckRemote(containerId, server)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify check.sh at location %s in container: %s on host: %s: %w", core.SAD_CHECK_SCRIPT_LOCATION, containerId, server.Host, err)
		}
	}

	return containerId, checkHash, nil
}

func deployInstanceFromCompose(instanceID, challengeName string, config *cfg.BeastChallengeConfig, stagingDir string, serverDeployed string, ports map[string]uint32) (string, string, error) {
	// Instanced compose challenges are managed by the projectName
	projectName := utils.ComposeDockerProjectNameInstanced(challengeName, instanceID)

	if ports[config.Challenge.Env.DefaultPortVar] != core.SSH_PORT {
		return "", "", fmt.Errorf("the default port variable does not map to %d for instance challenge %s", core.SSH_PORT, challengeName)
	}

	var err error
	var checkHash string
	var containerId string

	if cfg.Cfg.UseLocalDockerDaemon(serverDeployed) {
		containerId, err = cr.DeployContainerFromCompose(challengeName, projectName, stagingDir, config.Challenge.Env.DockerCompose, ports)
		if err != nil {
			return "", "", fmt.Errorf("failed to deploy instance %s: %w", instanceID, err)
		}

		err = verifySSHLocal(containerId)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify exposes of port %v in container %s on localhost", core.SSH_PORT, containerId)
		}

		checkHash, err = verifyCheckLocal(containerId)
		if err != nil {
			return "", "", fmt.Errorf("failed to deploy instance %s: %w", instanceID, err)
		}

		err = verifySSHLocal(containerId)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify exposes of port %v in container %s on localhost", core.SSH_PORT, containerId)
		}

		checkHash, err = verifyCheckLocal(containerId)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify check.sh at location %s in container: %s on localhost: %w", core.SAD_CHECK_SCRIPT_LOCATION, containerId, err)
		}
	} else {
		server := cfg.Cfg.AvailableServers[serverDeployed]
		containerId, err = remoteManager.DeployContainerFromComposeRemote(challengeName, projectName, stagingDir, config.Challenge.Env.DockerCompose, server, ports)
		if err != nil {
			return "", "", fmt.Errorf("failed to deploy compose on remote: %w", err)
		}

		err = verifySSHRemote(containerId, server)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify exposes of port %v in container %s on host %s", core.SSH_PORT, containerId, server.Host)
		}

		checkHash, err = verifyCheckRemote(containerId, server)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify check.sh at location %s in container: %s on host: %s: %w", core.SAD_CHECK_SCRIPT_LOCATION, containerId, server.Host, err)
		}
	}

	return containerId, checkHash, nil
}

func killInstanceContainer(containerID, deploymentType, instanceID, challengeName, serverDeployed string) error {
	if cfg.Cfg.UseLocalDockerDaemon(serverDeployed) {
		if deploymentType == core.DEPLOYMENT_TYPES["docker_compose"] {
			projectName := utils.ComposeDockerProjectNameInstanced(challengeName, instanceID)

			// managed by the projectName
			err := cr.ComposePurgeProject(projectName)
			if err != nil {
				return fmt.Errorf("docker compose down failed: %s", err.Error())
			}
		} else {
			// managed by the containerID
			err := cr.StopAndRemoveContainer(containerID)
			if err != nil {
				return fmt.Errorf("failed to stop container: %w", err)
			}
		}
	} else {
		server := cfg.Cfg.AvailableServers[serverDeployed]
		if deploymentType == core.DEPLOYMENT_TYPES["docker_compose"] {
			projectName := utils.ComposeDockerProjectNameInstanced(challengeName, instanceID)

			// managed by the projectName
			err := remoteManager.ComposePurgeProjectRemote(projectName, server)
			if err != nil {
				return fmt.Errorf("failed to stop compose on remote: %w", err)
			}
		} else {
			// managed by the containerID
			err := remoteManager.StopAndRemoveContainerRemote(containerID, server)
			if err != nil {
				return fmt.Errorf("failed to stop container on remote: %w", err)
			}
		}
	}

	return nil
}
