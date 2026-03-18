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
	if challenge.ID == 0 {
		return nil, fmt.Errorf("challenge not found: %s", challengeName)
	}

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	configFile := filepath.Join(stagingDir, core.CHALLENGE_CONFIG_FILE_NAME)

	var config cfg.BeastChallengeConfig
	_, err = toml.DecodeFile(configFile, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to load challenge config: %w", err)
	}

	if !config.Challenge.Metadata.IsInstanced() {
		return nil, fmt.Errorf("challenge %s is not configured for instancing", challengeName)
	}

	if challenge.ImageId == "" && config.Challenge.Env.DockerCompose == "" {
		return nil, fmt.Errorf("challenge %s has not been committed (no image available)", challengeName)
	}

	serverDeployed := selectServerForInstance()

	port, err := allocateInstancePort(serverDeployed)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate port: %w", err)
	}

	instanceID := uuid.New().String()[:12]

	expirationSeconds := config.Challenge.Metadata.GetInstanceExpiration()
	ttl := time.Duration(expirationSeconds) * time.Second
	expiresAt := time.Now().Add(ttl)

	var checkHash string
	var containerID string
	var deploymentType string

	if config.Challenge.Env.DockerCompose != "" {
		containerID, checkHash, err = deployInstanceFromCompose(instanceID, challengeName, port, &config, stagingDir, serverDeployed)
		deploymentType = core.DEPLOYMENT_TYPES["docker_compose"]
	} else {
		containerID, checkHash, err = deployInstanceContainer(instanceID, challengeName, port, challenge.ImageId, &config, serverDeployed)
		deploymentType = core.DEPLOYMENT_TYPES["standard_docker"]
	}

	if serverDeployed == "" || serverDeployed == core.LOCALHOST {
		err = addUserSSHKeyLocal(containerID, userSSHKey)
		if err != nil {
			return nil, fmt.Errorf("failed to add user ssh key: %w", err)
		}
	} else {
		err = addUserSSHKeyRemote(containerID, cfg.Cfg.AvailableServers[serverDeployed], userSSHKey)
		if err != nil {
			return nil, fmt.Errorf("failed to add user ssh key: %w", err)
		}
	}

	if err != nil {
		if err := cache.FreeContainerPorts(serverDeployed, containerID); err != nil {
			return nil, fmt.Errorf("failed to free container ports: %w", err)
		}
		return nil, fmt.Errorf("failed to deploy instance container: %w", err)
	}

	err = cache.RegisterFreePort(serverDeployed, containerID, port)
	if err != nil {
		log.Warnf("Failed to register port %d for container %s: %v", port, containerID, err)
	}

	instance := &cache.Instance{
		InstanceID:     instanceID,
		ChallengeName:  challengeName,
		ContainerID:    containerID,
		HostedAddress:  getHostedAddress(serverDeployed),
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
		if err := cache.FreeContainerPorts(serverDeployed, containerID); err != nil {
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

	err = cache.FreeContainerPorts(instance.ServerDeployed, instance.ContainerID)
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

func allocateInstancePort(host string) (uint32, error) {
	var firstPort, lastPort uint32
	var err error

	if host == core.LOCALHOST || host == "" {
		firstPort, lastPort, err = utils.ParsePortMapping(cfg.Cfg.LocalHostPortRange)
	} else {
		/* This is the simplest, not the best, solution to this...
		essentially selectServer returns the host, which is not necessarily the key used by AvailableServers so...
		a better solution would be to treat localhost explicitly as a server so it can be generalised
		(will also remove a lot of conditions scattered throughout the place) */
		var server *cfg.AvailableServer
		for _, s := range cfg.Cfg.AvailableServers {
			if s.Host == host {
				server = &s
				break
			}
		}

		if server == nil {
			return 0, fmt.Errorf("no available server found for host %s", host)
		}

		firstPort, lastPort, err = utils.ParsePortMapping(server.PortRange)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to parse port range: %w", err)
	}

	portRange := lastPort - firstPort + 1
	port, err := cache.GetFreePort(host, firstPort, portRange)
	if err != nil {
		return 0, fmt.Errorf("failed to allocate port: %w", err)
	}

	return port, nil
}

func freeInstancePort(host string, containerID string) {
	err := cache.FreeContainerPorts(host, containerID)
	if err != nil {
		log.Warnf("Failed to free ports for container %s on %s: %v", containerID, host, err)
	}
}

func selectServerForInstance() string {
	availableServer, err := remoteManager.ServerQueue.GetNextAvailableInstance()
	if err == nil && availableServer.Host != "" {
		return availableServer.Host
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

	sshAgentCheckCommand := fmt.Sprintf("[ -z \"$SSH_AUTH_SOCK\" ]")
	result, err := cr.RunCommandInContainer(containerId, []string{
		"sh", "-c", sshAgentCheckCommand,
	})

	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("ssh agent check failed, ssh-agent not running")
	}

	return nil
}

func verifySSHRemote(containerId string, server cfg.AvailableServer) error {
	portCmd := fmt.Sprintf("docker port %s %v/tcp", containerId, core.SSH_PORT)
	_, err := remoteManager.RunCommandOnServer(server, portCmd)
	if err != nil {
		return err
	}

	sshAgentCheckCommand := fmt.Sprintf("[ -z \"$SSH_AUTH_SOCK\" ]")
	result, err := remoteManager.RunCommandInContainerOnServer(server, containerId, sshAgentCheckCommand)

	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("ssh agent check failed, ssh-agent not running")
	}

	return err
}

func addUserSSHKeyLocal(containerId string, sshKey string) error {
	sshCmd := fmt.Sprintf("mkdir -p /home/beast/.ssh && echo '%s' >> /home/beast/.ssh/authorized_keys && chmod 700 /home/beast/.ssh && chmod 600 /home/beast/.ssh/authorized_keys && chown -R beast:beast /home/beast/.ssh", sshKey)

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
	sshCmd := fmt.Sprintf("mkdir -p /home/beast/.ssh && echo '%s' >> /home/beast/.ssh/authorized_keys && chmod 700 /home/beast/.ssh && chmod 600 /home/beast/.ssh/authorized_keys && chown -R beast:beast /home/beast/.ssh", sshKey)

	result, err := remoteManager.RunCommandInContainerOnServer(server, containerId, sshCmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to add public key to container: %s", result.Output)
	}

	return nil
}

func deployInstanceContainer(instanceID, challengeName string, hostPort uint32, imageID string, config *cfg.BeastChallengeConfig, serverDeployed string) (string, string, error) {
	containerName := fmt.Sprintf("beast_instance_%s_%s", challengeName, instanceID)

	containerPort := config.Challenge.Env.DefaultPort
	if containerPort != core.SSH_PORT {
		log.Warnln(fmt.Sprintf("Challenge %s does not have default port set to 22", challengeName))
		containerPort = 22
	}

	portMapping := []cr.PortMapping{
		{
			HostPort:      hostPort,
			ContainerPort: containerPort,
		},
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

	if serverDeployed == core.LOCALHOST || serverDeployed == "" {
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

func deployInstanceFromCompose(instanceID, challengeName string, hostPort uint32, config *cfg.BeastChallengeConfig, stagingDir string, serverDeployed string) (string, string, error) {
	projectName := fmt.Sprintf("beast-instance-%s-%s", coreUtils.EncodeID(challengeName), instanceID)
	composeFile := filepath.Join(stagingDir, challengeName, config.Challenge.Env.DockerCompose)

	var err error

	var checkHash string
	var containerId string

	if serverDeployed == core.LOCALHOST || serverDeployed == "" {
		err = utils.ValidateFileExists(composeFile)
		if err != nil {
			return "", "", fmt.Errorf("compose file not found: %w", err)
		}

		upCmd := exec.Command("docker", "compose",
			"-f", composeFile,
			"-p", projectName,
			"up", "-d")

		upCmd.Env = append(upCmd.Environ(), fmt.Sprintf("INSTANCE_PORT=%d", hostPort))

		var upOutput bytes.Buffer
		upCmd.Stdout = &upOutput
		upCmd.Stderr = &upOutput

		if err = upCmd.Run(); err != nil {
			log.Errorf("docker compose up failed for instance %s. Output:\n%s", instanceID, upOutput.String())
			return "", "", fmt.Errorf("docker compose up failed: %v", err)
		}

		psCmd := exec.Command("docker", "compose", "-p", projectName, "ps", "-q")
		var output bytes.Buffer
		psCmd.Stdout = &output

		if err = psCmd.Run(); err != nil {
			return "", "", fmt.Errorf("failed to get container IDs: %v", err)
		}

		containerIds := strings.Fields(strings.TrimSpace(output.String()))
		if len(containerIds) == 0 {
			return "", "", fmt.Errorf("no containers found for instance")
		}

		containerId = containerIds[0]
		if len(containerId) >= 12 {
			containerId = containerId[:12]
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
		containerId, err = remoteManager.DeployContainerFromComposeRemote(challengeName, stagingDir, config.Challenge.Env.DockerCompose, server)
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
	if serverDeployed == core.LOCALHOST || serverDeployed == "" {
		if deploymentType == core.DEPLOYMENT_TYPES["docker_compose"] {
			projectName := fmt.Sprintf("beast-instance-%s-%s", coreUtils.EncodeID(challengeName), instanceID)
			downCmd := exec.Command("docker", "compose", "-p", projectName, "down", "--remove-orphans", "-v")

			var output bytes.Buffer
			downCmd.Stdout = &output
			downCmd.Stderr = &output

			if err := downCmd.Run(); err != nil {
				return fmt.Errorf("docker compose down failed: %v, output: %s", err, output.String())
			}
		} else {
			err := cr.StopAndRemoveContainer(containerID)
			if err != nil {
				return fmt.Errorf("failed to stop container: %w", err)
			}
		}
	} else {
		server := cfg.Cfg.AvailableServers[serverDeployed]
		if deploymentType == core.DEPLOYMENT_TYPES["docker_compose"] {
			stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
			err := remoteManager.ComposePurgeRemote(challengeName, stagingDir, server)
			if err != nil {
				return fmt.Errorf("failed to stop compose on remote: %w", err)
			}
		} else {
			err := remoteManager.StopAndRemoveContainerRemote(containerID, server)
			if err != nil {
				return fmt.Errorf("failed to stop container on remote: %w", err)
			}
		}
	}

	return nil
}

func getHostedAddress(serverDeployed string) string {
	if serverDeployed != "" && serverDeployed != core.LOCALHOST {
		return serverDeployed
	}
	if cfg.Cfg.BeastStaticUrl != "" {
		return cfg.Cfg.BeastStaticUrl
	}
	return "localhost"
}
