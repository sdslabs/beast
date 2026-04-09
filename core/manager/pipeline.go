package manager

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sdslabs/beastv4/core/cache"

	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/notify"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	"github.com/sdslabs/beastv4/utils"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

// Run the staging step for the pipeline, this functions assumes the
// directory of the challenge wihch will be staged.
func stageChallenge(challengeDir string, config *cfg.BeastChallengeConfig) error {
	log.Debug("Starting staging stage of deploy pipeline")
	contextDir, err := getContextDirPath(challengeDir)
	if err != nil {
		return err
	}
	challengeName := filepath.Base(contextDir)

	log.Debugf("Found context directory for deploy : %s", contextDir)
	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	if err = utils.CreateIfNotExistDir(stagingDir); err != nil {
		return err
	}

	log.Debugf("Staging challenge to directory : %s", stagingDir)

	challengeConfig := filepath.Join(contextDir, core.CHALLENGE_CONFIG_FILE_NAME)
	log.Debugf("Reading challenge config from : %s", challengeConfig)

	var dockerfileCtx, serviceConfig, dockerCompose string
	dockerfileProvided := false

	additionalCtx := make(map[string]string)
	if config.Challenge.Env.DockerCompose != "" {
		dockerCompose = filepath.Join(challengeDir, config.Challenge.Env.DockerCompose)
		err := utils.ValidateFileExists(dockerCompose)
		if err != nil {
			return err
		}
		additionalCtx["docker-compose.yml"] = dockerCompose
		log.Debug("Got docker-compose file from the challenge config")
	} else {

		if config.Challenge.Env.DockerCtx != "" {
			dockerfileProvided = true
			dockerfileCtx = filepath.Join(challengeDir, config.Challenge.Env.DockerCtx)
			err := utils.ValidateFileExists(dockerfileCtx)
			if err != nil {
				return err
			}
			log.Debug("Got dockerfile context from the challenge config")
		} else {
			config.Challenge.Env.DockerCtx = core.DEFAULT_DOCKER_FILE
			dockerfileCtx, err = GenerateChallengeDockerfileCtx(config)
			if err != nil {
				return err
			}
			log.Debug("Generated dockerfile context for the challenge")
		}

		if config.Challenge.Metadata.Type == core.SERVICE_CHALLENGE_TYPE_NAME {
			if config.Challenge.Env.XinetdConf != "" {
				serviceConfig = filepath.Join(challengeDir, config.Challenge.Env.XinetdConf)
				err := utils.ValidateFileExists(serviceConfig)
				if err != nil {
					return err
				}
				additionalCtx[core.DEFAULT_XINETD_CONF_FILE] = serviceConfig
			}
		}
		additionalCtx["Dockerfile"] = dockerfileCtx
	}

	// Here we try to add all the additional context that are required like xinetd.conf
	// instead of mounting these files inside the container, since we want reproducibility
	// in the docker build if we provide the tar file to author himself. Embedding these
	// files inside the tar itself will make the tar build to be reproducible anywhere.
	err = appendAdditionalFileContexts(additionalCtx, config)
	if err != nil {
		return fmt.Errorf("error while adding additional context : %s", err)
	}

	// Copy those additional contexts to the staging area, so we can provide them for
	// user to download.
	copyAdditionalContextToStaging(additionalCtx, stagingDir)

	log.Debug("Copying Content to Static Folder")

	staticContentDir, err := GetStaticContentDir(challengeConfig, contextDir)
	if err != nil {
		return err
	}

	err = CopyToStaticContent(challengeName, staticContentDir)
	if err != nil {
		return err
	}

	log.Debug("Starting to build Tar file for the challenge to stage")

	if dockerfileProvided {
		delete(additionalCtx, "Dockerfile")
	}
	err = utils.Tar(contextDir, utils.Gzip, stagingDir, additionalCtx, []string{staticContentDir, filepath.Join(contextDir, core.HIDDEN)})
	if err != nil {
		return err
	}

	log.Debugf("Copying challenge config to staging directory")
	err = utils.CopyFile(challengeConfig, filepath.Join(stagingDir, core.CHALLENGE_CONFIG_FILE_NAME))
	if err != nil {
		return fmt.Errorf("error while copying challenge config to staging : %s", err)
	}

	log.Debugf("Staging for challenge %s complete", filepath.Base(challengeDir))
	return nil
}

// Commit the challenge as a docker image removing the previously existing image
// This first checks if there is an existing image for the challenge that exist
// if it exists then first the new image is created and then the old image is removed.
//
// stagedPath is the complete path to the tar file for the challenge in the staging dir
func commitChallenge(challenge *database.Challenge, config cfg.BeastChallengeConfig, stagedPath string, noCache bool) error {
	challengeName := config.Challenge.Metadata.Name
	challengeStagingDir := filepath.Dir(stagedPath)

	log.Debug("Starting commit stage for the challenge")
	err := utils.ValidateFileExists(stagedPath)
	if err != nil {
		return err
	}

	err = coreUtils.CleanupChallengeIfExist(config)
	if err != nil {
		log.Errorf("Error while cleaning up the challenge")
		return err
	}
	var (
		imageId  string
		buildErr error
		logBytes []byte
	)
	imageId = ""

	challengeTag := coreUtils.EncodeID(challengeName)
	log.Printf("== Server for challenge %s : %s", challengeName, challenge.ServerDeployed)
	if config.Challenge.Env.DockerCompose != "" {

		//  Should add some validation for the compose file

		if challenge.ServerDeployed != core.LOCALHOST && challenge.ServerDeployed != "" {

			server := cfg.Cfg.AvailableServers[challenge.ServerDeployed]
			logBytes, buildErr = remoteManager.BuildImagesFromComposeRemote(
				challengeName,
				challengeTag,
				stagedPath,
				server,
				noCache,
			)
		} else {
			var buff *bytes.Buffer

			buff, buildErr = cr.BuildImagesFromCompose(challengeName, challengeTag, stagedPath, config.Challenge.Env.DockerCompose, noCache)
			if buff != nil {
				logBytes = buff.Bytes()
			} else {
				logBytes = []byte("BuildImagesFromCompose returned nil buffer")
			}
		}
		// For Docker Compose challenges, ensure ImageId is empty in the database
		if err := database.UpdateChallenge(challenge, map[string]any{"ImageId": ""}); err != nil {
			return fmt.Errorf("error while setting empty ImageId for Docker Compose challenge: %s", err)
		}
	} else {
		if challenge.ServerDeployed != core.LOCALHOST && challenge.ServerDeployed != "" {
			server := cfg.Cfg.AvailableServers[challenge.ServerDeployed]
			stagedRemoteChallengePath := filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
			remoteStagedPath := filepath.Join(stagedRemoteChallengePath, fmt.Sprintf("%s.tar.gz", challengeName))
			err := remoteManager.ValidateFileRemoteExists(server, stagedRemoteChallengePath)
			if err != nil {
				return fmt.Errorf("error while checking if the challenge is staged on the remote server")
			}
			logBytes, imageId, buildErr = remoteManager.BuildImageFromTarContextRemote(challengeName, challengeTag, remoteStagedPath, server)
		} else {
			var buff *bytes.Buffer
			buff, imageId, buildErr = cr.BuildImageFromTarContext(challengeName, challengeTag, stagedPath, config.Challenge.Env.DockerCtx, noCache)
			if buff != nil {
				logBytes = buff.Bytes()
			} else {
				logBytes = []byte("BuildImageFromTarContext returned nil buffer")
			}
		}
	}
	// Create logs directory for the challenge in staging directory.
	challengeStagingLogsDir := filepath.Join(challengeStagingDir, core.BEAST_CHALLENGE_LOGS_DIR)
	err = utils.CreateIfNotExistDir(challengeStagingLogsDir)
	if err != nil {
		log.Errorf("Could not create challenge logs directory : %s : %s", challengeStagingLogsDir, err)
	} else if logBytes != nil {
		logFilePath := filepath.Join(challengeStagingLogsDir, fmt.Sprintf("%s.%s.log", challengeName, time.Now().Format("20060102150405")))
		logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
		if err != nil {
			log.Errorf("Error while writing logs to file : %s", logFilePath)
			return fmt.Errorf("error logs generated on image build failure could not be written to the logfile")
		}
		defer logFile.Close()

		logFile.Write(logBytes)
		log.Debug("Logs written to log file for the challenge")
	}

	if buildErr != nil {
		log.Error("Error while building image from the tar context of challenge")
		return buildErr
	}
	// Only update imageId for non-Docker Compose challenges
	if config.Challenge.Env.DockerCompose == "" {
		if imageId == "" {
			log.Error("Error while creating image logs written to the logfile")
			return fmt.Errorf("error while getting imageId for the commited challenge")
		}

		if err = database.UpdateChallenge(challenge, map[string]interface{}{"ImageId": imageId}); err != nil {
			return fmt.Errorf("error while writing imageId to database : %s", err)
		}
	}

	log.Infof("Image build for `%s` done", challengeName)

	return nil
}

// Deploy the challenge as a docker container from the image built
// This function first collects the environment variables and
// container config including ports, networks, resource limitations needed to spawn the container,
// then it creates the container and finally the challenge is deployed
//
// This function assumes that you have validated the configuration beforehand, so it won't be
// validated here.
func deployChallenge(challenge *database.Challenge, config cfg.BeastChallengeConfig) error {
	log.Debug("Starting to deploy the challenge")

	challengeName := config.Challenge.Metadata.Name
	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)

	if config.Challenge.Env.DockerCompose != "" {
		// currently the first container id returned
		var primaryContainerId string
		var err error

		composeFileName := config.Challenge.Env.DockerCompose
		portVariables, err := utils.ExtractPortsFromCompose(filepath.Join(stagingDir, challengeName, composeFileName))

		if err != nil {
			return fmt.Errorf("failed to extract port variables: %w", err)
		}

		serverDeployed := challenge.ServerDeployed
		if serverDeployed == "" {
			serverDeployed = core.LOCALHOST
		}

		ports := make(map[string]uint32, len(portVariables))
		for _, portVariable := range portVariables {
			port, err := allocateInstancePort(serverDeployed)
			if err != nil {
				return fmt.Errorf("failed to allocate instancePort: %w", err)
			}

			ports[portVariable] = port
		}

		if serverDeployed != core.LOCALHOST {
			server := cfg.Cfg.AvailableServers[challenge.ServerDeployed]
			/* Challenge Name and Project Name are the same for non instanced challenges */
			primaryContainerId, err = remoteManager.DeployContainerFromComposeRemote(challengeName, challengeName, stagingDir, composeFileName, server, ports)
		} else {
			/* Challenge Name and Project Name are the same for non instanced challenges */
			primaryContainerId, err = cr.DeployContainerFromCompose(challengeName, challengeName, stagingDir, composeFileName, ports)
		}

		if err != nil {
			for _, port := range ports {
				if err := cache.FreePortOnHost(serverDeployed, port); err != nil {
					log.Errorf("failed to free allocated compose port %d on host %s: %v", port, serverDeployed, err)
				}
			}
			return fmt.Errorf("error while deploying challenge with docker-compose on remote: %v", err)
		}

		for _, port := range ports {
			err = cache.AssignFreePortOnHostToContainer(serverDeployed, primaryContainerId, port)
			if err != nil {
				log.Warnf("Failed to register port %d for container %s: %v", port, primaryContainerId, err)
			}
		}

		// only for backward compatibility
		if err := database.UpdateChallenge(challenge, map[string]any{
			"ContainerId":    primaryContainerId,
			"DeploymentType": core.DEPLOYMENT_TYPES["docker_compose"],
		}); err != nil {
			return fmt.Errorf("error while updating Docker Compose challenge metadata: %s", err)
		}
		log.Infof("Challenge %s deployed with docker-compose successfully (primary container: %s)", challengeName, primaryContainerId)
		return nil
	}

	staticMount := make(map[string]string)
	var staticMountDir string
	if challenge.ServerDeployed == core.LOCALHOST || challenge.ServerDeployed == "" {
		staticMountDir = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, config.Challenge.Metadata.Name, core.BEAST_STATIC_FOLDER)
	} else {
		staticMountDir = filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR, config.Challenge.Metadata.Name, core.BEAST_STATIC_FOLDER)
	}
	relativeStaticContentDir := config.Challenge.Env.StaticContentDir
	if relativeStaticContentDir == "" {
		relativeStaticContentDir = core.PUBLIC
	}
	staticMount[staticMountDir] = filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, relativeStaticContentDir)
	log.Debugf("Static mount config for deploy : %s", staticMount)

	var containerEnv []string
	var containerNetwork string

	for _, env := range config.Challenge.Env.EnvironmentVars {
		containerEnv = append(containerEnv, fmt.Sprintf("%s=%s", env.Key, filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, env.Value)))
	}

	log.Debugf("Container config for challenge %s are: CPU(%d), Memory(%d), PidsLimit(%d)",
		config.Challenge.Metadata.Name,
		config.Resources.CPUShares,
		config.Resources.Memory,
		config.Resources.PidsLimit,
	)

	var err error
	host := challenge.ServerDeployed
	if host == "" {
		host = core.LOCALHOST
	}

	var firstPort, lastPort uint32
	server := cfg.Cfg.AvailableServers[host]
	firstPort, lastPort, err = utils.ParsePortMapping(server.PortRange)

	if err != nil {
		return fmt.Errorf("error while allocating ports on server %s for challenge %s: %s", host, challenge.Name, err.Error())
	}

	/* both ports are inclusive */
	portRange := lastPort - firstPort + 1

	ports := config.Challenge.Env.Ports
	portMapping := make([]cr.PortMapping, len(ports))

	for i, containerPort := range ports {
		hostPort, err := cache.GetFreePortOnHost(host, firstPort, portRange)
		if err != nil {
			return fmt.Errorf("error while getting free port on host %s: %s", host, err)
		}

		portMapping[i] = cr.PortMapping{
			HostPort:      hostPort,
			ContainerPort: containerPort,
		}
	}

	containerConfig := cr.CreateContainerConfig{
		PortMapping:      portMapping,
		MountsMap:        staticMount,
		ImageId:          challenge.ImageId,
		ContainerName:    coreUtils.EncodeID(config.Challenge.Metadata.Name),
		ContainerEnv:     containerEnv,
		ContainerNetwork: containerNetwork,
		Traffic:          config.Challenge.Env.TrafficType(),
		CPUShares:        config.Resources.CPUShares,
		Memory:           config.Resources.Memory,
		PidsLimit:        config.Resources.PidsLimit,
	}
	log.Debugf("create container config for challenge(%s): %v", config.Challenge.Metadata.Name, containerConfig)
	var containerId string
	if challenge.ServerDeployed == core.LOCALHOST || challenge.ServerDeployed == "" {
		containerId, err = cr.CreateContainerFromImage(&containerConfig)
	} else {
		server := cfg.Cfg.AvailableServers[challenge.ServerDeployed]
		containerId, err = remoteManager.CreateContainerFromImageRemote(containerConfig, server)
	}

	if err != nil {
		return fmt.Errorf("error while creating container for challenge %s: %s", challenge.Name, err.Error())
	}

	for _, portMap := range portMapping {
		if err := cache.AssignFreePortOnHostToContainer(host, containerId, portMap.HostPort); err != nil {
			return fmt.Errorf("error while registering port %v on host %s: %s", portMap.HostPort, host, err)
		}
	}

	if err = database.UpdateChallenge(challenge, map[string]any{
		"ContainerId":    containerId,
		"DeploymentType": core.DEPLOYMENT_TYPES["standard_docker"],
	}); err != nil {
		return fmt.Errorf("error while saving containerId to database : %s", err)
	}
	return nil
}

// This is the main function which starts the deploy pipeline for a locally
// available challenge, it goes through all the stages of the challenge deployement
// and hanles any error by logging into database if it occurs.
//
// challengeDir corresponds to the directory to be used as a challenge context
// For local challenge deployments this can be any directory.
//
// The pipeline goes through the following stages:
//
//   - stageChallenge - Add the challenge to the staging area for beast creating
//     a tar for the challenge with Dockerfile embedded into the context.
//     This challenge is then present in the staging area($BEAST_HOME/staging/challengeId/)
//     for further steps in the pipeline.
//
// The skipStage flag is a boolean value to skip the staging step for the challenge
// if this flag is true then the deployment to succeed the challenge should already
// be staged.
//
// If you are skipping the stage step make sure that you provide the challenge
// directory as the staged challenge directory, which contains the challenge config.
//
// During the staging steup if any error occurs, then the state of the challenge
// in the database is set to undeployed.
func bootstrapDeployPipeline(challengeDir string, skipStage bool, skipCommit bool, noCache bool) error {
	log.Debug("Loading Beast config")

	// If we are skipping commit step then we are automatically skipping
	// staging step.
	if skipCommit {
		skipStage = true
	}

	challengeName := filepath.Base(challengeDir)
	configFile := filepath.Join(challengeDir, core.CHALLENGE_CONFIG_FILE_NAME)

	var config cfg.BeastChallengeConfig
	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Errorf("Error while loading beast config for challenge %s : %s", challengeName, err)
		return fmt.Errorf("CONFIG ERROR: %s : %s", challengeName, err)
	}

	if !skipStage {
		err = config.ValidateRequiredFields(challengeDir)
		if err != nil {
			return fmt.Errorf("an error occured while validating the config file : %s, cannot continue with pipeline", err)
		}
	}

	// Validate challenge directory name with the name of the challenge
	// provided in the config file for the beast. There should be no
	// conflict in the name.
	if challengeName != config.Challenge.Metadata.Name {
		log.Errorf("Name of the challenge directory(%s) should match the name provided in the config file(%s)",
			challengeName,
			config.Challenge.Metadata.Name)
		return fmt.Errorf("CONFIG ERROR: %s : Inconsistent configuration name and challengeName", challengeName)
	}

	challenge, err := database.QueryFirstChallengeEntry("name", config.Challenge.Metadata.Name)
	if err != nil {
		log.Errorf("Error while querying challenge %s : %s", config.Challenge.Metadata.Name, err)
		return fmt.Errorf("DB ERROR: %s : %s", challengeName, err)
	}

	// Using the challenge dir we got, update the database entries for the challenge.
	err = UpdateOrCreateChallengeDbEntry(&challenge, config, "")
	if err != nil {
		log.Errorf("An error occured while creating db entry for challenge :: %s", challengeName)
		log.Errorf("Db error : %s", err)
		return fmt.Errorf("DB ERROR: %s : %s", challengeName, err)
	}

	// Check if the challenge type is static, if it is traditional deploy pipeline would not
	// follow, rather we would follow a static challenge deploy pipeline.
	if config.Challenge.Metadata.Type == core.STATIC_CHALLENGE_TYPE_NAME {
		if !skipStage {
			// Deploy pipeline for static challenge will follow.
			log.Infof("Deploy static challenge request.")
			DeployStaticChallenge(&config, &challenge, challengeDir)
		}
		return nil
	}

	// Look into the database to check if the deploy is already in progress
	// or not, return if a deploy is already in progress or else continue
	// deploying
	if challenge.Status != core.DEPLOY_STATUS["undeployed"] &&
		challenge.Status != core.DEPLOY_STATUS["deployed"] &&
		challenge.Status != core.DEPLOY_STATUS["queued"] &&
		challenge.Status != "" {
		log.Errorf("Deploy for %s already in progress, wait and check for the status(cur: %s)", challengeName, challenge.Status)
		return fmt.Errorf("PIPELINE START ERROR: %s : Deploy already in progress. Current Status : %s", challengeName, challenge.Status)
	}

	log.Debugf("Starting deploy pipeline for challenge %s", challengeName)

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	stagedChallengePath := filepath.Join(stagingDir, fmt.Sprintf("%s.tar.gz", challengeName))
	stagedRemoteChallengePath := filepath.Join(core.BEAST_REMOTE_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, fmt.Sprintf("%s.tar.gz", challengeName))
	if !skipStage {
		database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["staging"]})

		err = stageChallenge(challengeDir, &config)
		if err != nil {
			log.WithFields(log.Fields{
				"DEPLOY_ERROR": "STAGING :: " + challengeName,
			}).Errorf("%s", err)

			database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["undeployed"]})
			return fmt.Errorf("STAGING ERROR: %s : %s", challengeName, err)
		}
		if challenge.ServerDeployed != core.LOCALHOST && challenge.ServerDeployed != "" {
			remoteManager.StageChallRemote(cfg.Cfg.AvailableServers[challenge.ServerDeployed], challenge)
		}
	} else {
		log.Debugf("Checking if challenge already staged")
		if challenge.ServerDeployed != core.LOCALHOST && challenge.ServerDeployed != "" {
			err := remoteManager.ValidateFileRemoteExists(cfg.Cfg.AvailableServers[challenge.ServerDeployed], stagedRemoteChallengePath)
			if err != nil {
				msg := "Challenge not already in staged(but skipping asked), could not proceed further"
				log.WithFields(log.Fields{
					"DEPLOY_ERROR": "STAGING :: " + challengeName,
				}).Errorf("%s", msg)

				return fmt.Errorf("STAGING ERROR: %s : %s", challengeName, msg)
			}
		} else {
			err = utils.ValidateFileExists(stagedChallengePath)
			if err != nil {
				msg := "Challenge not already in staged(but skipping asked), could not proceed further"
				log.WithFields(log.Fields{
					"DEPLOY_ERROR": "STAGING :: " + challengeName,
				}).Errorf("%s", msg)

				return fmt.Errorf("STAGING ERROR: %s : %s", challengeName, msg)
			}
		}
		log.Infof("SKIPPING STAGING STEP IN THE DEPLOY PIPELINE")
	}

	if !skipCommit {
		database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["committing"]})

		err = commitChallenge(&challenge, config, stagedChallengePath, noCache)
		if err != nil {
			log.WithFields(log.Fields{
				"DEPLOY_ERROR": "COMMIT :: " + challengeName,
			}).Errorf("%s", err)

			database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["undeployed"]})
			return fmt.Errorf("COMMIT ERROR: %s : %s", challengeName, err)
		}

	} else {
		if challenge.ImageId == "" {
			database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["undeployed"]})
			return fmt.Errorf("COMMIT ERROR: Cannot skip commit step, no Image ID found for challenge.")
		}
		log.Debugf("Skipping commit phase")
	}

	if challenge.Instanced {
		database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["deployed"]})
		log.Infof("Challenge %s is instanced, skipping deploy stage", challengeName)
		return nil
	}

	database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["deploying"]})

	err = deployChallenge(&challenge, config)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "DEPLOY :: " + challengeName,
		}).Errorf("%s", err)

		database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["undeployed"]})

		return fmt.Errorf("DEPLOY ERROR: %s : %s", challengeName, err)
	}

	database.UpdateChallenge(&challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["deployed"]})

	log.Infof("CHALLENGE %s DEPLOYED SUCCESSFULLY", challengeName)

	return nil
}

// This is just a decorator function over bootstrapDeployPipeline and generate
// notifications to slack on the basis of the result of the deploy pipeline.
func StartDeployPipeline(challengeDir string, skipStage bool, skipCommit bool, noCache bool) {
	challengeName := filepath.Base(challengeDir)
	var sendNotificationError error

	err := bootstrapDeployPipeline(challengeDir, skipStage, skipCommit, noCache)
	if err != nil {
		sendNotificationError = notify.SendNotification(notify.Error, err.Error())
	} else {
		msg := fmt.Sprintf("DEPLOY SUCCESS : %s : Challenge deployment pipeline successful.", challengeName)
		sendNotificationError = notify.SendNotification(notify.Success, msg)
	}

	if sendNotificationError == nil {
		log.Debugf("%s: Notification sent", challengeName)
	}
}
