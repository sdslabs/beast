package deploy

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

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

	log.Debugf("Found context directory for deploy : %s", contextDir)
	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
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
func CommitChallenge(stagedPath, challengeName string) error {
	log.Debug("Starting commit stage for the challenge")
	err := utils.ValidateFileExists(stagedPath)
	if err != nil {
		return err
	}

	// TODO: Implement client.ImageSearch() here to check if the image
	// already exist first check the database and then
	// the docker images.
	builderContext, err := os.Open(stagedPath)
	defer builderContext.Close()
	if err != nil {
		log.Errorf("Error while opening staged file for the challenge")
		return fmt.Errorf("Error while opening staged file :: %s", stagedPath)
	}

	ctx := context.Background()
	buildOptions := types.ImageBuildOptions{
		Tags: []string{challengeName, "latest"},
	}

	imageBuildResp, err := DockerClient.ImageBuild(ctx, builderContext, buildOptions)
	if err != nil {
		log.Errorf("An error while build image for challenge %s :: %s", challengeName, err)
		return err
	}
	defer imageBuildResp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(imageBuildResp.Body)

	fmt.Println(buf)
	log.Debug("Image build in process")

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

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	stagedChallengePath := filepath.Join(stagingDir, fmt.Sprintf("%s.tar.gz", challengeName))
	err = CommitChallenge(stagedChallengePath, challengeName)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "COMMIT :: " + challengeName,
		}).Errorf("%s", err)
		return
	}
}

func init() {
	log.Info("Trying to connect to docker client for beast")

	DockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Error while creating a docker client for beast: %s", err)
	}

	log.Infof("Using docker client version %s", DockerClient.ClientVersion())
}
