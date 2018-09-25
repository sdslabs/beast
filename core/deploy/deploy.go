package deploy

import (
	log "github.com/sirupsen/logrus"
)

func DeployChallenge(challengeDir string) error {
	log.Infof("Deploying Challenge : %s", challengeDir)

	if err := ValidateChallengeDir(challengeDir); err != nil {
		return err
	}

	// StartDeployPipeline(challengeDir)
	return nil
}
