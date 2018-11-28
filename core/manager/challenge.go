package manager

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/database"
	"github.com/sdslabs/beastv4/docker"
	"github.com/sdslabs/beastv4/git"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

// Main function which starts the deploy of a challenge provided
// directory inside the hack git database. We validate the challenge
// config first and return early starting a goroutine to start out
// the deploy process. The early return consist of validation of the
// provided challenge config in the directory.
func DeployChallengePipeline(challengeDir string) error {
	log.Infof("Deploying Challenge : %s", challengeDir)

	if err := ValidateChallengeDir(challengeDir); err != nil {
		log.Errorf("Error validating the challenge directory %s : %s", challengeDir, err)
		return err
	}

	// Start a goroutine to start a deploy pipeline for the challenge
	go StartDeployPipeline(challengeDir, false)

	return nil
}

// Start deploying a challenge using the challenge Name(we are not using ID here),
// if the challenge is already present
// and the container is running, then don't do anything. If the challenge does not exist
// then first check if the challenge is in staged state, if it is then deploy challenge
// from there on or else start deploy pipeline for the challenge.
func DeployChallenge(challengeName string) error {
	log.Infof("Processing request to deploy the challenge with ID %s", challengeName)

	challenge, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		log.Errorf("Got an error while querying database for challenge : %s : %s", challengeName, err)
		return errors.New("DATABASE SERVER ERROR")
	}

	// Check if a container for the challenge is already deployed.
	// If the challange is already deployed, return an error.
	// If not then start the deploy pipeline for the challenge.
	if challenge.ContainerId != "" {
		containers, err := docker.SearchContainerByFilter(map[string]string{"id": challenge.ContainerId})
		if err != nil {
			log.Error("Error while searching for container with id %s", challenge.ContainerId)
			return errors.New("DOCKER ERROR")
		}

		if len(containers) > 1 {
			log.Error("Got more than one containers, something fishy here. Check manually")
			return errors.New("DOCKER ERROR")
		}

		if len(containers) == 1 {
			log.Debugf("Found an already running instance of the challenge with container ID %s", challenge.ContainerId)
			return fmt.Errorf("Challenge already deployed")
		} else {
			if err = database.Db.Model(&challenge).Update("ContainerId", "").Error; err != nil {
				log.Errorf("Error while saving challenge state in database : %s", err)
				return errors.New("DATABASE ERROR")
			}
		}
	}

	challengeStagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	// TODO: Later replace this with a manifest file, containing information about the
	// staged challenge. Currently this staging will only check for non static challenges
	// so static challenges will be redeployed each time. Later we can improve this by adding this
	// test to the manifest file.
	stagedFileName := filepath.Join(challengeStagingDir, fmt.Sprintf("%s.tar.gz", challengeName))
	log.Infof("No challenge exists with the provided challenge name, starting deploy for new instance")

	// Check if the challenge is in staged state, it it is start the
	// pipeline from there on, else start deploy pipeline for the challenge
	// from remote
	err = utils.ValidateFileExists(stagedFileName)
	if err != nil {
		log.Infof("The requested challenge with Name %s is not already staged", challengeName)
		challengeDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, config.Cfg.GitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR, challengeName)

		if err := ValidateChallengeDir(challengeDir); err != nil {
			log.Errorf("Error validating the challenge directory %s : %s", challengeDir, err)
			return errors.New("CHALLENGE VALIDATION ERROR")
		}

		go StartDeployPipeline(challengeDir, false)
	} else {
		// Challenge is in staged state, so start the deploy pipeline and skip
		// the staging state.
		log.Infof("The requested challenge with Name %s is already staged, starting deploy...", challengeName)
		go StartDeployPipeline(challengeStagingDir, true)
	}

	return nil
}

// Deploy multiple challenges simultaneously.
// When we have multiple challenges we spawn X goroutines and distribute
// deployments in those goroutines. The work for these worker goroutines is specified
// in deployList, which contains the name of the challenges to be deployed.
func DeployMultipleChallenges(deployList []string) {
	deployList = utils.GetUniqueStrings(deployList)
	log.Infof("Starting deploy for the following challenge list : %v", deployList)

	for _, chall := range deployList {
		log.Infof("Starting to push %s challenge to deploy queue", chall)
		// TODO: Discuss if to make this challenge force redeploy or not.
		err := DeployChallenge(chall)
		if err != nil {
			log.Errorf("Cannot start deploy for challenge : %s due to : %s", chall, err)
			continue
		}

		log.Infof("Started deploy for challenge : %s", chall)
	}
}

// Deploy all challenges.
func DeployAll(sync bool) error {
	log.Infof("Got request to deploy ALL CHALLENGES")
	if sync {
		err := git.SyncBeastRemote()
		if err != nil {
			// A hack for go-git which returns error when the git repo
			// is up to date. This ignores this error.
			if !strings.Contains(err.Error(), "already up-to-date") {
				log.Warnf("Error while syncing beast for DEPLOY_ALL : %s ...", err)
				return fmt.Errorf("GIT_REMOTE_SYNC_ERROR")
			}
		}

		log.Debugf("Sync for beast remote done for DEPLOY_ALL")
	}

	challengesDirRoot := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, config.Cfg.GitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR)
	err, challenges := utils.GetDirsInDir(challengesDirRoot)
	if err != nil {
		log.Errorf("DEPLOY_ALL : Error while getting available challenges : %s", err)
		return fmt.Errorf("DIRECTORY_ACCESS_ERROR")
	}

	var challsNameList []string
	for _, chall := range challenges {
		challsNameList = append(challsNameList, chall)
	}

	go DeployMultipleChallenges(challsNameList)
	return nil
}

// Undeploy a challenge, remove the container for the challenge in question
// update the database entries for the challenge.
// Do not touch any files in staging, commit phase.
// This function returns a error if the challenge was not found or if
// an error happened while removing the challenge instance.
func UndeployChallenge(challengeName string) error {
	log.Infof("Got request to Undeploy challenge : %s", challengeName)

	challenge, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		log.Errorf("Got an error while querying database for challenge : %s : %s", challengeName, err)
		return errors.New("DATABASE SERVER ERROR")
	}

	if challenge.Name == "" {
		log.Errorf("Invalid challenge Name for undeploy action")
		return fmt.Errorf("ChallengeName %s not valid", challengeName)
	}

	// If a existing container ID is not found make sure that you atleast
	// set the deploy status to unknown. This earlier caused problem since if a challenge
	// was in staging state(and deployed is cancled) then we can neither deploy new
	// version nor we can undeploy the existing version(since it does not exist)
	// So this....
	if challenge.ContainerId == "" {
		log.Warnf("No instance of challenge(%s) deployed", challengeName)
		database.Db.Model(&challenge).Update("Status", core.DEPLOY_STATUS["unknown"])
		return fmt.Errorf("No instance of challenge(%s) deployed", challengeName)
	}

	log.Debug("Removing challenge instance for ", challengeName)
	err = docker.StopAndRemoveContainer(challenge.ContainerId)
	if err != nil {
		p := fmt.Errorf("Error while removing challenge instance : %s", err)
		log.Error(p.Error())
		return p
	}

	database.Db.Model(&challenge).Updates(map[string]interface{}{
		"Status":      core.DEPLOY_STATUS["unknown"],
		"ContainerId": "",
		"ImageId":     "",
	})

	if err != nil {
		log.Error(err)
		return fmt.Errorf("Error while updating the challenge : %s", err)
	}

	log.Infof("Challenge undeploy successful for %s", challenge.Name)

	return nil
}
