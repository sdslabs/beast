package deploy

import (
	log "github.com/sirupsen/logrus"
)

// Main function which starts the deploy of a challenge provided
// directory inside the hack git database.
func DeployChallenge(challengeDir string) error {
	log.Infof("Deploying Challenge : %s", challengeDir)

	if err := ValidateChallengeDir(challengeDir); err != nil {
		return err
	}

	// Start a goroutine to start a deploy pipeline for the challenge
	go StartDeployPipeline(challengeDir)

	return nil
}
