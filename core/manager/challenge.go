package manager

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/database"
	"github.com/sdslabs/beastv4/docker"
	"github.com/sdslabs/beastv4/git"
	"github.com/sdslabs/beastv4/notify"
	"github.com/sdslabs/beastv4/utils"

	"github.com/BurntSushi/toml"
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
	challengeName := filepath.Base(challengeDir)
	info := DeployInfo{
		ChallDir:   challengeDir,
		SkipStage:  false,
		SkipCommit: false,
	}
	Q.CheckPush(Work{
		Action:    core.MANAGE_ACTION_DEPLOY,
		ChallName: challengeName,
		Info:      info,
	})
	return nil
}

// Start deploying a challenge using the challenge Name(we are not using ID here),
// if the challenge is already present
// and the container is running, then don't do anything. If the challenge does not exist
// then first check if the challenge is in staged state, if it is then deploy challenge
// from there on or else start deploy pipeline for the challenge.
//
// This will start deploy pipeline if it finds there is no problem in deployment, else it will
// notify the user via return value if there is an error or if the deployement request cannot
// be processed.
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
	if utils.IsContainerIdValid(challenge.ContainerId) {
		containers, err := docker.SearchContainerByFilter(map[string]string{"id": challenge.ContainerId})
		if err != nil {
			log.Error("Error while searching for container with id %s", challenge.ContainerId)
			return errors.New("DOCKER ERROR")
		}

		if len(containers) > 1 {
			log.Error("Got more than one containers, something fishy here. Contact admin to check manually.")
			return errors.New("DOCKER ERROR")
		}

		if len(containers) == 1 {
			log.Debugf("Found an already running instance of the challenge with container ID %s", challenge.ContainerId)
			return fmt.Errorf("Challenge already deployed")
		} else {
			if err = database.UpdateChallenge(&challenge, map[string]interface{}{"ContainerId": utils.GetTempContainerId(challengeName)}); err != nil {
				log.Errorf("Error while saving challenge state in database : %s", err)
				return errors.New("DATABASE ERROR")
			}
		}
	}

	challengeStagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)

	if utils.IsImageIdValid(challenge.ImageId) {
		imageExist, err := docker.CheckIfImageExists(challenge.ImageId)
		if err != nil {
			log.Errorf("Error while searching for image with id %s: %s", challenge.ImageId, err)
			return errors.New("DOCKER ERROR")
		}

		if imageExist {
			log.Debugf("Found a commited instance of the challenge with image ID %s", challenge.ImageId)
			log.Debugf("Challenge is already in commited stage, deploying from existing image.")
			// Challenge is already in commited stage here, so skip commit and stage step and start
			// deployment of the challenge.
			info := DeployInfo{
				ChallDir:   challengeStagingDir,
				SkipStage:  true,
				SkipCommit: true,
			}
			return Q.CheckPush(Work{
				Action:    core.MANAGE_ACTION_DEPLOY,
				ChallName: challengeName,
				Info:      info,
			})
		} else {
			if err = database.UpdateChallenge(&challenge, map[string]interface{}{"ImageId": utils.GetTempImageId(challengeName)}); err != nil {
				log.Errorf("Error while saving challenge state in database : %s", err)
				return errors.New("DATABASE ERROR")
			}
		}
	}

	// TODO: Later replace this with a manifest file, containing Information about the
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

		info := DeployInfo{
			ChallDir:   challengeDir,
			SkipStage:  false,
			SkipCommit: false,
		}
		return Q.CheckPush(Work{
			Action:    core.MANAGE_ACTION_DEPLOY,
			ChallName: challengeName,
			Info:      info,
		})
	} else {
		// Challenge is in staged state, so start the deploy pipeline and skip
		// the staging state.
		log.Infof("The requested challenge with Name %s is already staged, starting deploy...", challengeName)

		info := DeployInfo{
			ChallDir:   challengeStagingDir,
			SkipStage:  true,
			SkipCommit: false,
		}
		return Q.CheckPush(Work{
			Action:    core.MANAGE_ACTION_DEPLOY,
			ChallName: challengeName,
			Info:      info,
		})
	}
}

// Deploy multiple challenges simultaneously.
// When we have multiple challenges we spawn X goroutines and distribute
// deployments in those goroutines. The work for these worker goroutines is specified
// in deployList, which contains the name of the challenges to be deployed.
func DeployMultipleChallenges(deployList []string) []string {
	deployList = utils.GetUniqueStrings(deployList)
	log.Infof("Starting deploy for the following challenge list : %v", deployList)

	errstrings := []string{}

	for _, chall := range deployList {
		log.Infof("Starting to push %s challenge to deploy queue", chall)
		// TODO: Discuss if to make this challenge force redeploy or not.
		err := DeployChallenge(chall)
		if err != nil {
			log.Errorf("Cannot start deploy for challenge : %s due to : %s", chall, err)
			errstrings = append(errstrings, err.Error())
			continue
		}

		log.Infof("Started deploy for challenge : %s", chall)
	}
	return errstrings
}

// Deploy all challenges.
func DeployAll(sync bool) []string {
	log.Infof("Got request to deploy ALL CHALLENGES")
	if sync {
		err := git.SyncBeastRemote()
		if err != nil {
			// A hack for go-git which returns error when the git repo
			// is up to date. This ignores this error.
			if !strings.Contains(err.Error(), "already up-to-date") {
				log.Warnf("Error while syncing beast for DEPLOY_ALL : %s ...", err)
				return []string{fmt.Sprintf("GIT_REMOTE_SYNC_ERROR")}
			}
		}

		log.Debugf("Sync for beast remote done for DEPLOY_ALL")
	}

	challengesDirRoot := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, config.Cfg.GitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR)
	err, challenges := utils.GetDirsInDir(challengesDirRoot)
	if err != nil {
		log.Errorf("DEPLOY_ALL : Error while getting available challenges : %s", err)
		return []string{fmt.Sprintf("DIRECTORY_ACCESS_ERROR")}
	}

	var challsNameList []string
	for _, chall := range challenges {
		challsNameList = append(challsNameList, chall)
	}

	return DeployMultipleChallenges(challsNameList)
}

// Unstage a challenge based on the challenge name.
// This simply removes the staging directory for the challenge from the staging area.
func unstageChallenge(challengeName string) error {
	challengeStagedDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)
	err := utils.ValidateDirExists(challengeStagedDir)
	if err != nil {
		log.Warnf("Challenge staging directory for challenge %s does not exist, continuing...", challengeName)
		return nil
	}

	err = utils.RemoveDirRecursively(challengeStagedDir)
	if err != nil {
		return fmt.Errorf("Error while removing staged directory : %s", err)
	}

	return nil
}

// Undeploy a challenge, remove the container for the challenge in question
// update the database entries for the challenge.
// Do not touch any files in staging, commit phase.
// This function returns a error if the challenge was not found or if
// an error happened while removing the challenge instance.
func undeployChallenge(challengeName string, purge bool) error {
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
	if challenge.ContainerId == utils.GetTempContainerId(challengeName) {
		log.Warnf("No instance of challenge(%s) deployed", challengeName)
	} else {
		log.Debug("Removing challenge instance for ", challengeName)
		err = docker.StopAndRemoveContainer(challenge.ContainerId)
		if err != nil {
			// This should not return from here, this should assume that
			// the container instance does not exist and hence should update the database
			// with the container ID.
			p := fmt.Errorf("Error while removing challenge instance : %s", err)
			log.Error(p.Error())
		}
	}

	tx := database.UpdateChallenge(&challenge, map[string]interface{}{
		"Status":      core.DEPLOY_STATUS["unknown"],
		"ContainerId": utils.GetTempContainerId(challengeName),
	})

	if tx.Error != nil {
		log.Error(tx.Error)
		return fmt.Errorf("Error while updating the challenge : %s", tx.Error)
	}

	log.Infof("Challenge undeploy successful for %s", challenge.Name)

	// If purge is true then first cleanup the challenge image and container
	// and then remove the challenge from the staging directory.
	if purge {
		configFile := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, config.Cfg.GitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR, challengeName, core.CHALLENGE_CONFIG_FILE_NAME)
		var cfg config.BeastChallengeConfig
		_, err = toml.DecodeFile(configFile, &cfg)
		if err != nil {
			return err
		}

		err = coreUtils.CleanupChallengeIfExist(cfg)
		if err != nil {
			return fmt.Errorf("Error while cleaning up the challenge: %s", err)
		}

		log.Infof("Purging the challenge : %s", challenge.Name)
		err = unstageChallenge(challenge.Name)
		if err != nil {
			return fmt.Errorf("Error while purging in unstage step: %s", err)
		}

		log.Infof("Challenge purge successful")
	}

	return nil
}

func StartUndeployChallenge(challengeName string, purge bool) error {
	err := undeployChallenge(challengeName, purge)
	if err != nil {
		msg := fmt.Sprintf("UNDEPLOY ERROR: %s : %s", challengeName, err)
		log.Error(msg)
		notify.SendNotificationToSlack(notify.Error, msg)
	} else {
		msg := fmt.Sprintf("UNDEPLOY SUCCESSFUL: %s", challengeName)
		log.Info(msg)
		notify.SendNotificationToSlack(notify.Success, msg)
	}

	log.Infof("Notification for the event sent to slack.")
	return err
}

func UndeployChallenge(challengeName string) error {
	return Q.CheckPush(Work{
		Action:    core.MANAGE_ACTION_UNDEPLOY,
		ChallName: challengeName,
	})
}

func PurgeChallenge(challengeName string) error {
	return Q.CheckPush(Work{
		Action:    core.MANAGE_ACTION_PURGE,
		ChallName: challengeName,
	})
}

func RedeployChallenge(challengeName string) error {
	return Q.CheckPush(Work{
		Action:    core.MANAGE_ACTION_REDEPLOY,
		ChallName: challengeName,
	})
}
