package manager

import (
	"fmt"
	"path/filepath"
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

func SpawnInstance(challengeName, userID, username string) (*cache.Instance, error) {
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

	instanceID := uuid.New().String()[:12]

	expirationSeconds := config.Challenge.Metadata.GetInstanceExpiration()
	ttl := time.Duration(expirationSeconds) * time.Second
	expiresAt := time.Now().Add(ttl)

	var port uint32
	var containerID string
	var deploymentType string

	if config.Challenge.Env.DockerCompose != "" {
		composeFile := filepath.Join(stagingDir, challengeName, config.Challenge.Env.DockerCompose)
		portVariables, err := utils.ExtractPortsFromCompose(composeFile)

		if err != nil {
			return nil, fmt.Errorf("failed to extract port variables: %w", err)
		}

		ports := make(map[string]uint32, len(portVariables))
		for _, portVariable := range portVariables {
			port, err = allocateInstancePort(serverDeployed)
			if err != nil {
				return nil, fmt.Errorf("failed to allocate instancePort: %w", err)
			}

			ports[portVariable] = port
		}

		if len(portVariables) > 0 {
			port = ports[portVariables[0]]
		}

		containerID, err = deployInstanceFromCompose(instanceID, challengeName, &config, stagingDir, serverDeployed, ports)
		deploymentType = core.DEPLOYMENT_TYPES["docker_compose"]

		if err != nil {
			for _, port := range ports {
				if err := cache.FreePortOnHost(serverDeployed, port); err != nil {
					log.Errorf("failed to free container ports: %s", err.Error())
				}
			}
			return nil, fmt.Errorf("failed to deploy instance container: %s", err.Error())
		}

		for _, port := range ports {
			err = cache.AssignFreePortOnHostToContainer(serverDeployed, containerID, port)
			if err != nil {
				log.Warnf("Failed to register port %d for container %s: %v", port, containerID, err)
			}
		}
	} else {
		port, err := allocateInstancePort(serverDeployed)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate port: %w", err)
		}

		containerID, err = deployInstanceContainer(instanceID, challengeName, port, challenge.ImageId, &config, serverDeployed)
		deploymentType = core.DEPLOYMENT_TYPES["standard_docker"]

		if err != nil {
			if err := cache.FreePortOnHost(serverDeployed, port); err != nil {
				return nil, fmt.Errorf("failed to free port %v: %w", port, err)
			}

			return nil, fmt.Errorf("failed to deploy instance container: %w", err)
		}

		err = cache.AssignFreePortOnHostToContainer(serverDeployed, containerID, port)
		if err != nil {
			log.Warnf("Failed to register port %d for container %s: %v", port, containerID, err)
		}
	}

	instance := &cache.Instance{
		InstanceID:     instanceID,
		ChallengeName:  challengeName,
		ContainerID:    containerID,
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
		if err := cache.FreeContainerPortsOnHost(serverDeployed, containerID); err != nil {
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

	err = cache.FreeContainerPortsOnHost(instance.ServerDeployed, instance.ContainerID)
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
		server := cfg.Cfg.AvailableServers[host]
		firstPort, lastPort, err = utils.ParsePortMapping(server.PortRange)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to parse port range: %w", err)
	}

	portRange := lastPort - firstPort + 1
	port, err := cache.GetFreePortOnHost(host, firstPort, portRange)
	if err != nil {
		return 0, fmt.Errorf("failed to allocate port: %w", err)
	}

	return port, nil
}

func selectServerForInstance() string {
	availableServer, err := remoteManager.ServerQueue.GetNextAvailableInstance()
	if err == nil && availableServer.Host != "" {
		return availableServer.Host
	}
	return core.LOCALHOST
}

func deployInstanceContainer(instanceID, challengeName string, hostPort uint32, imageID string, config *cfg.BeastChallengeConfig, serverDeployed string) (string, error) {
	containerName := fmt.Sprintf("beast_instance_%s_%s", challengeName, instanceID)

	containerPort := config.Challenge.Env.DefaultPort
	if containerPort == 0 {
		containerPort = 8080
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

	var containerId string
	var err error

	if serverDeployed == core.LOCALHOST || serverDeployed == "" {
		containerId, err = cr.CreateContainerFromImage(&containerConfig)
	} else {
		server := cfg.Cfg.AvailableServers[serverDeployed]
		containerId, err = remoteManager.CreateContainerFromImageRemote(containerConfig, server)
	}

	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return containerId, nil
}

func deployInstanceFromCompose(instanceID, challengeName string, config *cfg.BeastChallengeConfig, stagingDir string, serverDeployed string, ports map[string]uint32) (string, error) {
	projectName := fmt.Sprintf("instance-%s-%s", coreUtils.EncodeID(challengeName), instanceID)

	if serverDeployed == core.LOCALHOST || serverDeployed == "" {
		primaryContainer, err := cr.DeployContainerFromCompose(challengeName, projectName, stagingDir, config.Challenge.Env.DockerCompose, ports)
		if err != nil {
			return "", fmt.Errorf("failed to deploy instance %s: %w", instanceID, err)
		}

		return primaryContainer, nil
	} else {
		server := cfg.Cfg.AvailableServers[serverDeployed]
		containerId, err := remoteManager.DeployContainerFromComposeRemote(challengeName, projectName, stagingDir, config.Challenge.Env.DockerCompose, server, ports)
		if err != nil {
			return "", fmt.Errorf("failed to deploy compose on remote: %w", err)
		}

		return containerId, nil
	}
}

func killInstanceContainer(containerID, deploymentType, instanceID, challengeName, serverDeployed string) error {
	if serverDeployed == core.LOCALHOST || serverDeployed == "" {
		if deploymentType == core.DEPLOYMENT_TYPES["docker_compose"] {
			stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
			projectName := fmt.Sprintf("instance-%s-%s", coreUtils.EncodeID(challengeName), instanceID)

			err := cr.ComposePurge(projectName, stagingDir)
			if err != nil {
				return fmt.Errorf("docker compose down failed: %s", err.Error())
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
