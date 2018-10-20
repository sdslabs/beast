package manager

import (
	"errors"
	"fmt"

	"github.com/sdslabs/beastv4/database"
	"github.com/sdslabs/beastv4/docker"

	log "github.com/sirupsen/logrus"
)

// Main function which starts the deploy of a challenge provided
// directory inside the hack git database.
func DeployChallengePipeline(challengeDir string) error {
	log.Infof("Deploying Challenge : %s", challengeDir)

	if err := ValidateChallengeDir(challengeDir); err != nil {
		log.Errorf("Error validating the challenge directory %s : %s", challengeDir, err)
		return err
	}

	// Start a goroutine to start a deploy pipeline for the challenge
	go StartDeployPipeline(challengeDir)

	return nil
}

// Undeploy a challenge, remove the container for the challenge in question
// update the database entries for the challenge.
// Do not touch any files in staging, commit phase.
// This function returns a error if the challenge was not found or if
// an error happened while removing the challenge instance.
func UndeployChallenge(challengeId string) error {
	log.Infof("Got request to Undeploy challenge : %s", challengeId)

	challenge, err := database.QueryFirstChallengeEntry("challenge_id", challengeId)
	if err != nil {
		log.Errorf("Got an error while querying database for challenge : %s : %s", challengeId, err)
		return errors.New("DATABASE SERVER ERROR")
	}

	if challenge.ChallengeId == "" {
		log.Errorf("Invalid challengeID for undeploy action")
		return fmt.Errorf("ChallengeId %s not valid", challengeId)
	}

	if challenge.ContainerId == "" {
		log.Errorf("No instance of challenge(%s) deployed", challengeId)
		return fmt.Errorf("No instance of challenge(%s) deployed", challengeId)
	}

	log.Debug("Removing challenge instance for ", challengeId)
	err = docker.StopAndRemoveContainer(challenge.ContainerId)
	if err != nil {
		p := fmt.Errorf("Error while removing challenge instance : %s", err)
		log.Error(p.Error())
		return p
	}

	log.Infof("Challenge undeploy successful for %s", challenge.Name)

	return nil
}
