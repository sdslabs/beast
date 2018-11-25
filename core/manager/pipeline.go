package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/database"
	"github.com/sdslabs/beastv4/docker"
	"github.com/sdslabs/beastv4/utils"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

// Run the staging setp for the pipeline, this functions assumes the
// directory of the challenge wihch will be staged.
func stageChallenge(challengeDir string) error {
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

	dockerfileCtx, err := GenerateChallengeDockerfileCtx(challengeConfig)
	if err != nil {
		return err
	}
	log.Debug("Got dockerfile context from the challenge config")

	additionalCtx := make(map[string]string)
	additionalCtx["Dockerfile"] = dockerfileCtx

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
	err = utils.Tar(contextDir, utils.Gzip, stagingDir, additionalCtx, []string{staticContentDir})

	if err != nil {
		return err
	}

	log.Debugf("Copying challenge config to staging directory")
	err = utils.CopyFile(challengeConfig, filepath.Join(stagingDir, core.CHALLENGE_CONFIG_FILE_NAME))
	if err != nil {
		return fmt.Errorf("Error while copying challenge config to staging : %s", err)
	}

	log.Debugf("Staging for challenge %s complete", filepath.Base(challengeDir))
	return nil
}

// Commit the challenge as a docker image removing the previously existing image
// This first checks if there is an existing image for the challenge that exist
// if it exists then first the new image is created and then the old image is removed.
//
// stagedPath is the complete path to the tar file for the challenge in the staging dir
func commitChallenge(challenge *database.Challenge, config cfg.BeastChallengeConfig, stagedPath string) error {
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

	buff, imageId, err := docker.BuildImageFromTarContext(challengeName, stagedPath)
	if err != nil {
		log.Error("Error while building image from the tar context of challenge")
		return err
	}

	if imageId == "" {
		log.Error("Could not figure out the ImageID for the commited challenge")
		return fmt.Errorf("Error while getting imageId for the commited challenge")
	}

	challenge.ImageId = imageId
	if err = database.Db.Save(challenge).Error; err != nil {
		return fmt.Errorf("Error while writing imageId to database : %s", err)
	}

	log.Debug("Writing image build logs from buffer to file")

	challengeStagingLogsDir := filepath.Join(challengeStagingDir, core.BEAST_CHALLENGE_LOGS_DIR)
	err = utils.CreateIfNotExistDir(challengeStagingLogsDir)
	if err != nil {
		log.Errorf("Could not validate challenge logs directory : %s : %s", challengeStagingLogsDir, err)
		return nil
	}

	logFilePath := filepath.Join(challengeStagingLogsDir, fmt.Sprintf("%s.%s.log", challengeName, time.Now().Format("20060102150405")))
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		log.Errorf("Error while writing logs to file : %s", logFilePath)
	}
	defer logFile.Close()

	logFile.Write(buff.Bytes())
	log.Debug("Logs writterned to log file for the challenge")

	log.Infof("Image build for `%s` done", challengeName)

	return nil
}

func deployChallenge(challenge *database.Challenge, config cfg.BeastChallengeConfig) error {
	log.Debug("Starting to deploy the challenge")

	containerId, err := docker.CreateContainerFromImage(config.Challenge.Env.Ports, nil, challenge.ImageId, config.Challenge.Metadata.Name)
	if err != nil {
		if containerId != "" {
			challenge.ContainerId = containerId
			if e := database.Db.Save(challenge).Error; e != nil {
				return fmt.Errorf("Error while starting container : %s and saving database : %s", err, e)
			}

			return fmt.Errorf("Error while starting the container : %s", err)
		}

		return fmt.Errorf("Error while trying to create a container for the challenge: %s", err)
	}

	challenge.ContainerId = containerId
	if err = database.Db.Save(challenge).Error; err != nil {
		return fmt.Errorf("Error while saving containerId to database : %s", err)
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
// * stageChallenge - Add the challenge to the staging area for beast creating
//		a tar for the challenge with Dockerfile embedded into the context.
// 		This challenge is then present in the staging area($BEAST_HOME/staging/challengeId/)
//		for further steps in the pipeline.
//
// The skipStage flag is a boolean value to skip the staging step for the challenge
// if this flag is true then the deployment to succeed the challenge should already
// be staged.
//
// If you are skipping the stage step make sure that you provide the challenge
// directory as the staged challenge directory, which contains the challenge config.
//
// During the staging steup if any error occurs, then the state of the challenge
// in the database is set to unknown.
func StartDeployPipeline(challengeDir string, skipStage bool) {
	log.Debug("Loading Beast config")

	challengeName := filepath.Base(challengeDir)
	configFile := filepath.Join(challengeDir, core.CHALLENGE_CONFIG_FILE_NAME)

	var config cfg.BeastChallengeConfig
	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Errorf("Error while loading beast config for challenge %s : %s", challengeName, err)
		return
	}

	// We do not validate challenge config here, make sure you have validated the config
	// beforehand
	//
	// err = config.ValidateRequiredFields()
	// if err != nil {
	// 	log.Errorf("An error occured while validating the config file : %s", err)
	//	return
	//}

	// Validate challenge directory name with the name of the challenge
	// provided in the config file for the beast. THere should be no
	// conflict in the name.
	if challengeName != config.Challenge.Metadata.Name {
		log.Errorf("Name of the challenge directory(%s) should match the name provided in the config file(%s)", challengeName, config.Challenge.Metadata.Name)
		return
	}

	challenge, err := database.QueryFirstChallengeEntry("name", config.Challenge.Metadata.Name)
	if err != nil {
		log.Errorf("Error while querying challenge %s : %s", config.Challenge.Metadata.Name, err)
		return
	}

	// Using the challenge dir we got, update the database entries for the challenge.
	err = updateOrCreateChallengeDbEntry(&challenge, config)
	if err != nil {
		log.Errorf("An error occured while creating db entry for challenge :: %s", challengeName)
		log.Errorf("Db error : %s", err)
		return
	}

	// Check if the challenge type is static, if it is traditional deploy pipeline would not
	// follow, rather we would follow a static challenge deploy pipeline.
	if config.Challenge.Metadata.Type == core.STATIC_CHALLENGE_TYPE_NAME {
		if !skipStage {
			// Deploy pipeline for static challenge will follow.
			DeployStaticChallenge(&config)
		}
		return
	}
	// Look into the database to check if the deploy is already in progress
	// or not, return if a deploy is already in progress or else continue
	// deploying
	if challenge.Status != core.DEPLOY_STATUS["unknown"] &&
		challenge.Status != core.DEPLOY_STATUS["deployed"] {
		log.Errorf("Deploy for %s already in progress, wait and check for the status(cur: %s)", challengeName, challenge.Status)
		return
	}

	log.Debugf("Starting deploy pipeline for challenge %s", challengeName)

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	stagedChallengePath := filepath.Join(stagingDir, fmt.Sprintf("%s.tar.gz", challengeName))

	if !skipStage {
		challenge.Status = core.DEPLOY_STATUS["staging"]
		database.Db.Save(&challenge)

		err = stageChallenge(challengeDir)
		if err != nil {
			log.WithFields(log.Fields{
				"DEPLOY_ERROR": "STAGING :: " + challengeName,
			}).Errorf("%s", err)

			challenge.Status = core.DEPLOY_STATUS["unknown"]
			database.Db.Save(&challenge)
			return
		}
	} else {
		log.Debugf("Checking if challenge already staged")

		err = utils.ValidateFileExists(stagedChallengePath)
		if err != nil {
			log.WithFields(log.Fields{
				"DEPLOY_ERROR": "STAGING :: " + challengeName,
			}).Errorf("Challenge not already in staged(but skipping asked), could not proceed further")
			return
		}

		log.Infof("SKIPPING STAGING STEP IN THE DEPLOY PIPELINE")
	}

	challenge.Status = core.DEPLOY_STATUS["committing"]
	database.Db.Save(&challenge)

	err = commitChallenge(&challenge, config, stagedChallengePath)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "COMMIT :: " + challengeName,
		}).Errorf("%s", err)

		challenge.Status = core.DEPLOY_STATUS["unknown"]
		database.Db.Save(&challenge)
		return
	}

	challenge.Status = core.DEPLOY_STATUS["deploying"]
	database.Db.Save(&challenge)

	err = deployChallenge(&challenge, config)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "DEPLOY :: " + challengeName,
		}).Errorf("%s", err)

		challenge.Status = core.DEPLOY_STATUS["unknown"]
		database.Db.Save(&challenge)
		return
	}

	challenge.Status = core.DEPLOY_STATUS["deployed"]
	database.Db.Save(&challenge)

	log.Infof("CHALLENGE %s DEPLOYED SUCCESSFULLY", challengeName)
}
