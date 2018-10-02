package deploy

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fristonio/beast/core"
	"github.com/fristonio/beast/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var DockerClient client.Client

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
func CommitChallenge(stagedPath, challengeName string) error {
	challengeStagingDir := filepath.Dir(stagedPath)
	log.Debug("Starting commit stage for the challenge")
	err := utils.ValidateFileExists(stagedPath)
	if err != nil {
		return err
	}

	// TODO: Implement client.ImageSearch() here to check if the image
	// already exist first check the database and then
	// the docker images.
	builderContext, err := os.Open(stagedPath)
	if err != nil {
		log.Errorf("Error while opening staged file for the challenge")
		return fmt.Errorf("Error while opening staged file :: %s", stagedPath)
	}
	defer builderContext.Close()

	buildOptions := types.ImageBuildOptions{
		Tags: []string{challengeName, "latest"},
	}

	log.Debug("Connecting to docker daemon to build image")
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Errorf("Error while creating a docker client for beast: %s", err)
		return err
	}

	log.Debug("Image build in process")
	imageBuildResp, err := dockerClient.ImageBuild(context.Background(), builderContext, buildOptions)
	if err != nil {
		log.Errorf("An error while build image for challenge %s :: %s", challengeName, err)
		return err
	}
	defer imageBuildResp.Body.Close()

	// TODO: Add entry to database - Image Build is Done

	buf := new(bytes.Buffer)
	buf.ReadFrom(imageBuildResp.Body)

	log.Debug("Writing image build logs from buffer to file")
	logFilePath := filepath.Join(challengeStagingDir, fmt.Sprintf("%s.%s.log", challengeName, time.Now().Format("20060102150405")))
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		log.Errorf("Error while writing logs to file : %s", logFilePath)
	}
	defer logFile.Close()

	logFile.Write(buf.Bytes())
	log.Debug("Logs writterned to log file for the challenge")

	log.Infof("Image build for `%s` done", challengeName)

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
	challengeName := filepath.Base(challengeDir)
	log.Debugf("Starting deploy pipeline for challenge %s", challengeName)

	err := StageChallenge(challengeDir)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "STAGING :: " + challengeName,
		}).Errorf("%s", err)
		return
	}

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	stagedChallengePath := filepath.Join(stagingDir, fmt.Sprintf("%s.tar.gz", challengeName))
	err = CommitChallenge(stagedChallengePath, challengeName)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "COMMIT :: " + challengeName,
		}).Errorf("%s", err)
		return
	}
}
