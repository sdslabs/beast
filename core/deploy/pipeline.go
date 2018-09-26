package deploy

import (
	"path/filepath"

	"github.com/fristonio/beast/core"
	"github.com/fristonio/beast/utils"
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

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)

	challengeConfig := filepath.Join(contextDir, core.CONFIG_FILE_NAME)
	dockerfileCtx, err := GenerateChallengeDockerfileCtx(challengeConfig)
	if err != nil {
		return err
	}

	additionalCtx := make(map[string]string)
	additionalCtx["Dockerfile"] = dockerfileCtx

	err = utils.Tar(contextDir, utils.Gzip, stagingDir, additionalCtx)
	if err != nil {
		return err
	}

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
	log.Debug("Starting deploy pipeline for challenge %s", challengeName)

	err := StageChallenge(challengeDir)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "STAGING :: " + challengeName,
		}).Errorf("%s", err)
		return
	}
}
