package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/database"
	"github.com/sdslabs/beastv4/docker"
	"github.com/sdslabs/beastv4/utils"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

// Run the staging setp for the pipeline, this functions assumes the
// directory of the challenge wihch will be staged.
func StageChallenge(challengeDir string) error {
	log.Debug("Starting staging stage of deploy pipeline")
	contextDir, err := GetContextDirPath(challengeDir)
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

	challengeConfig := filepath.Join(contextDir, core.CONFIG_FILE_NAME)
	log.Debugf("Reading challenge config from : %s", challengeConfig)

	dockerfileCtx, err := GenerateChallengeDockerfileCtx(challengeConfig)
	if err != nil {
		return err
	}
	log.Debug("Got dockerfile context from the challenge config")

	additionalCtx := make(map[string]string)
	additionalCtx["Dockerfile"] = dockerfileCtx

	log.Debug("Starting to build Tar file for the challenge to stage")
	err = utils.Tar(contextDir, utils.Gzip, stagingDir, additionalCtx)
	if err != nil {
		return err
	}

	log.Debugf("Staging for challenge %s complete", filepath.Base(challengeDir))
	return nil
}

// Commit the challenge as a docker image removing the previously existing image
// This first checks if there is an existing image for the challenge that exist
// if it exists then first the new image is created and then the old image is removed.
//
// stagedPath is the complete path to the tar file for the challenge in the staging dir
func CommitChallenge(challenge *database.Challenge, config core.BeastConfig, stagedPath string) error {
	challengeName := config.Challenge.Name
	challengeStagingDir := filepath.Dir(stagedPath)

	log.Debug("Starting commit stage for the challenge")
	err := utils.ValidateFileExists(stagedPath)
	if err != nil {
		return err
	}

	err = core.CleanupChallengeIfExist(config)
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

	logFilePath := filepath.Join(challengeStagingDir, fmt.Sprintf("%s.%s.log", challengeName, time.Now().Format("20060102150405")))
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

func DeployChallenge(challengeId string) error {
	log.Debug("Starting to deploy the challenge")

	return nil
}

// This is the main function which starts the deploy pipeline for a locally
// available challenge, it goes through all the stages of the challenge deployement
// and hanles any error by logging into database if it occurs.
//
// challengeDir corresponds to the directory to be used as a challenge context
//
// The pipeline goes through the following stages:
// * StageChallenge - Add the challenge to the staging area for beast creating
//		a tar for the challenge with Dockerfile embedded into the context.
// 		This challenge is then present in the staging area($BEAST_HOME/staging/challengeId/)
//		for further steps in the pipeline.
func StartDeployPipeline(challengeDir string) {
	log.Debug("Loading Beast config")

	challengeName := filepath.Base(challengeDir)
	configFile := filepath.Join(challengeDir, core.CONFIG_FILE_NAME)

	var config core.BeastConfig
	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Errorf("Error while loading beast config for challenge %s : %s", challengeName, err)
		return
	}

	err = config.ValidateRequiredFields()
	if err != nil {
		log.Errorf("An error occured while validating the config file : %s", err)
		return
	}

	// Validate challenge directory name with the name of the challenge
	// provided in the config file for the beast. THere should be no
	// conflict in the name.
	if challengeName != config.Challenge.Name {
		log.Errorf("Name of the challenge directory(%s) should match the name provided in the config file(%s)", challengeName, config.Challenge.Name)
		return
	}

	log.Debugf("Starting deploy pipeline for challenge %s", challengeName)

	challenge, err := UpdateOrCreateChallengeDbEntry(config)
	if err != nil {
		log.Errorf("An error occured while creating db entry for challenge :: %s", challengeName)
		log.Errorf("Db error : %s", err)
		return
	}

	challenge.Status = core.DEPLOY_STATUS["stage"]
	database.Db.Save(&challenge)

	err = StageChallenge(challengeDir)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "STAGING :: " + challengeName,
		}).Errorf("%s", err)
		return
	}

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	stagedChallengePath := filepath.Join(stagingDir, fmt.Sprintf("%s.tar.gz", challengeName))

	challenge.Status = core.DEPLOY_STATUS["commit"]
	database.Db.Save(&challenge)

	err = CommitChallenge(&challenge, config, stagedChallengePath)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "COMMIT :: " + challengeName,
		}).Errorf("%s", err)
		return
	}

	challenge.Status = core.DEPLOY_STATUS["deploy"]
	database.Db.Save(&challenge)

	err = DeployChallenge(config.Challenge.Id)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "DEPLOY :: " + challengeName,
		}).Errorf("%s", err)
		return
	}
}
