package manager

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/notify"
	wpool "github.com/sdslabs/beastv4/pkg/workerpool"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

type TaskInfo struct {
	Action     string
	ChallDir   string
	SkipStage  bool
	SkipCommit bool
	Purge      bool
}

var Q *wpool.Queue

// a struct which implements the wpool.Worker interface for performing tasks
type Worker struct {
}

var ChallengeActionHandlers = map[string]func(string) error{
	core.MANAGE_ACTION_DEPLOY:   DeployChallenge,
	core.MANAGE_ACTION_UNDEPLOY: UndeployChallenge,
	core.MANAGE_ACTION_REDEPLOY: RedeployChallenge,
	core.MANAGE_ACTION_PURGE:    PurgeChallenge,
}

// Function which commits the deployed challenge provided
func CommitChallengeContainer(challName string) error {
	log.Debug("Starting to commit the chall : %s", challName)
	chall, err := database.QueryFirstChallengeEntry("name", challName)
	if err != nil {
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		return err
	}

	if chall.Status != core.DEPLOY_STATUS["deployed"] || !coreUtils.IsContainerIdValid(chall.ContainerId) {
		log.Errorf("Challenge : %s not deployed", err.Error())
		return fmt.Errorf("Challenge is not deployed")
	}

	imageId, err := cr.CommitContainer(chall.ContainerId)
	if err != nil {
		log.Errorf("Error while commiting the container : %s", err.Error())
		return err
	}
	imageId = strings.TrimPrefix(imageId, "sha256:")

	if e := database.UpdateChallenge(&chall, map[string]interface{}{"ImageId": imageId}); e != nil {
		log.Errorf("Error while updating imageid : %s", e.Error())
		return e
	}
	return nil
}

// This function is used by the worker nodes or goroutines to perform the task which is pushed in the queue by the beast manager
func (worker *Worker) PerformTask(w wpool.Task) *wpool.Task {
	info := w.Info.(TaskInfo)
	switch info.Action {
	case core.MANAGE_ACTION_DEPLOY:
		StartDeployPipeline(info.ChallDir, info.SkipStage, info.SkipCommit)

	case core.MANAGE_ACTION_UNDEPLOY:
		err := StartUndeployChallenge(w.ID, false)
		if err != nil {
			log.Errorf("Error while undeplying challenge(%s): %s", w.ID, err.Error())
		}

	case core.MANAGE_ACTION_REDEPLOY:
		err := StartUndeployChallenge(w.ID, true)
		if err != nil {
			log.Errorf("Error while redeplying challenge(%s): %s", w.ID, err.Error())
			return nil
		}
		work, err := GetDeployWork(w.ID)
		if err != nil {
			log.Error(err)
			return nil
		}
		return work

	case core.MANAGE_ACTION_PURGE:
		err := StartUndeployChallenge(w.ID, true)
		if err != nil {
			log.Errorf("Error while purging challenge(%s): %s", w.ID, err.Error())
		}

	default:
		chall, err := database.QueryFirstChallengeEntry("name", w.ID)
		if err != nil {
			log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		}
		database.UpdateChallenge(&chall, map[string]interface{}{"Status": core.DEPLOY_STATUS["unknown"]})
		log.Errorf("The action(%s) specified for challenge : %s does not exist", info.Action, w.ID)

	}

	return nil
}

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
	info := TaskInfo{
		Action:     core.MANAGE_ACTION_DEPLOY,
		ChallDir:   challengeDir,
		SkipStage:  false,
		SkipCommit: false,
	}

	chall, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		return err
	}

	database.UpdateChallenge(&chall, map[string]interface{}{"Status": core.DEPLOY_STATUS["queued"]})

	return Q.Push(wpool.Task{
		ID:   challengeName,
		Info: info,
	})
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
func GetDeployWork(challengeName string) (*wpool.Task, error) {
	log.Infof("Processing request to deploy the challenge with ID %s", challengeName)

	challenge, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		log.Errorf("Got an error while querying database for challenge : %s : %s", challengeName, err)
		return nil, errors.New("DATABASE SERVER ERROR")
	}

	// Check if a container for the challenge is already deployed.
	// If the challange is already deployed, return an error.
	// If not then start the deploy pipeline for the challenge.
	if coreUtils.IsContainerIdValid(challenge.ContainerId) {
		containers, err := cr.SearchContainerByFilter(map[string]string{"id": challenge.ContainerId})
		if err != nil {
			log.Error("Error while searching for container with id %s", challenge.ContainerId)
			return nil, errors.New("CONTAINER RUNTIME ERROR")
		}

		if len(containers) > 1 {
			log.Error("Got more than one containers, something fishy here. Contact admin to check manually.")
			return nil, errors.New("CONTAINER RUNTIME ERROR")
		}

		if len(containers) == 1 {
			log.Debugf("Found an already running instance of the challenge with container ID %s", challenge.ContainerId)
			return nil, fmt.Errorf("Challenge already deployed")
		} else {
			if err = database.UpdateChallenge(&challenge, map[string]interface{}{"ContainerId": coreUtils.GetTempContainerId(challengeName)}); err != nil {
				log.Errorf("Error while saving challenge state in database : %s", err)
				return nil, errors.New("DATABASE ERROR")
			}
		}
	}

	challengeStagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName)

	if coreUtils.IsImageIdValid(challenge.ImageId) {
		imageExist, err := cr.CheckIfImageExists(challenge.ImageId)
		if err != nil {
			log.Errorf("Error while searching for image with id %s: %s", challenge.ImageId, err)
			return nil, errors.New("CONTAINER RUNTIME ERROR")
		}

		if imageExist {
			log.Debugf("Found a commited instance of the challenge with image ID %s", challenge.ImageId)
			log.Debugf("Challenge is already in commited stage.")
			// Challenge is already in commited stage here, so skip commit and stage step and start
			// deployment of the challenge.
			log.Debugf("Checking and pushing the task of deploying commited challenge in the queue.")
			info := TaskInfo{
				Action:     core.MANAGE_ACTION_DEPLOY,
				ChallDir:   challengeStagingDir,
				SkipStage:  true,
				SkipCommit: true,
			}
			return &wpool.Task{
				ID:   challengeName,
				Info: info,
			}, nil
		} else {
			if err = database.UpdateChallenge(&challenge, map[string]interface{}{"ImageId": coreUtils.GetTempImageId(challengeName)}); err != nil {
				log.Errorf("Error while saving challenge state in database : %s", err)
				return nil, errors.New("DATABASE ERROR")
			}
		}
	}

	// TODO: Later replace this with a manifest file, containing Information about the
	// staged challenge. Currently this staging will only check for non static challenges
	// so static challenges will be redeployed each time. Later we can improve this by adding this
	// test to the manifest file.
	stagedFileName := filepath.Join(challengeStagingDir, fmt.Sprintf("%s.tar.gz", challengeName))
	log.Infof("No challenge exists with the provided challenge name.")

	// Check if the challenge is in staged state, it it is start the
	// pipeline from there on, else start deploy pipeline for the challenge
	// from remote
	err = utils.ValidateFileExists(stagedFileName)
	if err != nil {
		log.Infof("The requested challenge with Name %s is not already staged", challengeName)
		challengeDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, config.Cfg.GitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR, challengeName)

		/// TODO : remove multiple validation while deploying challenge
		if err := ValidateChallengeDir(challengeDir); err != nil {
			log.Errorf("Error validating the challenge directory %s : %s", challengeDir, err)
			return nil, errors.New("CHALLENGE VALIDATION ERROR")
		}

		log.Debugf("Checking and pushing the task of deploying unstaged challenge in the queue.")

		info := TaskInfo{
			Action:     core.MANAGE_ACTION_DEPLOY,
			ChallDir:   challengeDir,
			SkipStage:  false,
			SkipCommit: false,
		}
		return &wpool.Task{
			ID:   challengeName,
			Info: info,
		}, nil
	} else {
		// Challenge is in staged state, so start the deploy pipeline and skip
		// the staging state.
		log.Infof("The requested challenge with Name %s is already staged.", challengeName)

		log.Debugf("Checking and pushing the task of deploying staged challenge in the queue.")

		info := TaskInfo{
			Action:     core.MANAGE_ACTION_DEPLOY,
			ChallDir:   challengeStagingDir,
			SkipStage:  true,
			SkipCommit: false,
		}
		return &wpool.Task{
			ID:   challengeName,
			Info: info,
		}, nil
	}
}

// Handle multiple challenges simultaneously.
// When we have multiple challenges we spawn X goroutines and distribute
// deployments in those goroutines. The work for these worker goroutines is specified
// in list, which contains the name of the challenges.
func handleMultipleChallenges(list []string, action string) []string {
	list = utils.GetUniqueStrings(list)
	log.Infof("Starting %s for the following challenge list : %v", action, list)

	if len(list) == 0 {
		return []string{"EMPTY LIST"}
	}

	errstrings := []string{}

	challAction, ok := ChallengeActionHandlers[action]

	if !ok {
		return []string{"ACTION NOT IN LIST"}
	}

	for _, chall := range list {

		log.Infof("Starting to push %s challenge to queue", chall)

		err := challAction(chall)

		if err != nil {
			log.Errorf("Cannot start %s for challenge : %s due to : %s", action, chall, err)
			errstrings = append(errstrings, fmt.Sprintf("%s : %s", chall, err.Error()))
			continue
		}
		log.Infof("Started %s for challenge : %s", action, chall)
	}
	return errstrings
}

// Handle tag related challenges.
func HandleTagRelatedChallenges(action string, tag string, user string) []string {
	log.Infof("Trying request to %s CHALLENGES related to %s", action, tag)

	tagEntry := &database.Tag{
		TagName: tag,
	}

	// TODO: Why are we creating tag entry here, if there does not
	// exist the provided tag, simply skip doing anything.
	err := database.QueryOrCreateTagEntry(tagEntry)
	if err != nil {
		return []string{fmt.Sprintf("DATABASE_ERROR")}
	}

	challs, err := database.QueryRelatedChallenges(tagEntry)
	if err != nil {
		return []string{fmt.Sprintf("DATABASE_ERROR")}
	}

	var challsNameList []string

	err = appendAndSaveTransaction(&challs, &challsNameList, action, user)
	if err != nil {
		return []string{err.Error()}
	}

	return handleMultipleChallenges(challsNameList, action)
}

func appendAndSaveTransaction(challs *[]database.Challenge, challsNameList *[]string, action string, user string) error {
	author, err := database.QueryFirstAuthorEntry("name", user)
	if err != nil {
		return err
	}

	for _, chall := range *challs {
		*challsNameList = append(*challsNameList, chall.Name)
		TransactionEntry := database.Transaction{
			Action:      action,
			AuthorID:    author.ID,
			ChallengeID: chall.ID,
		}

		tran := database.SaveTransaction(&TransactionEntry)
		if tran != nil {
			log.Infof("Error while saving transaction : %s ", tran)
		}
	}
	return nil
}

// Handle all challenges.
func HandleAll(action string, user string) []string {
	log.Infof("Got request to %s ALL CHALLENGES", action)

	if action == core.MANAGE_ACTION_DEPLOY {
		err := SyncBeastRemote()
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

	var challsNameList []string
	var err error

	switch action {
	case core.MANAGE_ACTION_DEPLOY:
		challengesDirRoot := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, config.Cfg.GitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR)
		err, challenges := utils.GetDirsInDir(challengesDirRoot)
		if err != nil {
			break
		}
		for _, chall := range challenges {
			//TODO : challenge transaction save for deploying is not done since ID is not provided here
			challsNameList = append(challsNameList, chall)
		}

	case core.MANAGE_ACTION_UNDEPLOY:
		challenges, err := database.QueryChallengeEntriesMap(map[string]interface{}{
			"Status": core.DEPLOY_STATUS["deployed"],
		})
		if err != nil {
			break
		}

		err = appendAndSaveTransaction(&challenges, &challsNameList, action, user)

	case core.MANAGE_ACTION_REDEPLOY:
		challenges, err := database.QueryChallengeEntriesMap(map[string]interface{}{
			"Status": core.DEPLOY_STATUS["deployed"],
		})
		if err != nil {
			break
		}

		err = appendAndSaveTransaction(&challenges, &challsNameList, action, user)

	case core.MANAGE_ACTION_PURGE:
		challenges, err := database.QueryAllChallenges()
		if err != nil {
			break
		}

		err = appendAndSaveTransaction(&challenges, &challsNameList, action, user)
	}

	if err != nil {
		return []string{fmt.Sprintf("ACCESS_ERROR : %s", err.Error())}
	}
	return handleMultipleChallenges(challsNameList, action)
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
	if challenge.ContainerId == coreUtils.GetTempContainerId(challengeName) {
		log.Warnf("No instance of challenge(%s) deployed", challengeName)
	} else {
		log.Debug("Removing challenge instance for ", challengeName)
		err = cr.StopAndRemoveContainer(challenge.ContainerId)
		if err != nil {
			// This should not return from here, this should assume that
			// the container instance does not exist and hence should update the database
			// with the container ID.
			p := fmt.Errorf("Error while removing challenge instance : %s", err)
			log.Error(p.Error())
		}
	}

	err = database.UpdateChallenge(&challenge, map[string]interface{}{
		"Status":      core.DEPLOY_STATUS["unknown"],
		"ContainerId": coreUtils.GetTempContainerId(challengeName),
	})

	if err != nil {
		log.Error(err)
		return fmt.Errorf("Error while updating the challenge : %s", err)
	}

	log.Infof("Challenge undeploy successful for %s", challenge.Name)

	// If purge is true then first cleanup the challenge image and container
	// and then remove the challenge from the staging directory.
	if purge {
		configFile := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, core.CHALLENGE_CONFIG_FILE_NAME)
		var cfg config.BeastChallengeConfig
		_, err = toml.DecodeFile(configFile, &cfg)
		if err != nil {
			return err
		}
		err = cleanSidecar(&cfg)
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

func DeployChallenge(challengeName string) error {
	w, err := GetDeployWork(challengeName)
	if err != nil {
		return err
	}

	chall, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		return err
	}
	if chall.Name == "" {
		database.UpdateChallenge(&chall, map[string]interface{}{"Status": core.DEPLOY_STATUS["queued"]})
	}
	return Q.Push(*w)
}

func UndeployChallenge(challengeName string) error {

	chall, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		return err
	}

	if chall.Name == "" {
		database.UpdateChallenge(&chall, map[string]interface{}{"Status": core.DEPLOY_STATUS["queued"]})
	}

	return Q.Push(wpool.Task{
		Info: TaskInfo{Action: core.MANAGE_ACTION_UNDEPLOY},
		ID:   challengeName,
	})
}

func PurgeChallenge(challengeName string) error {

	chall, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		return err
	}
	if chall.Name == "" {
		database.UpdateChallenge(&chall, map[string]interface{}{"Status": core.DEPLOY_STATUS["queued"]})
	}

	return Q.Push(wpool.Task{
		Info: TaskInfo{Action: core.MANAGE_ACTION_PURGE},
		ID:   challengeName,
	})
}

func RedeployChallenge(challengeName string) error {

	chall, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err != nil {
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		return err
	}
	if chall.Name == "" {
		database.UpdateChallenge(&chall, map[string]interface{}{"Status": core.DEPLOY_STATUS["queued"]})
	}

	return Q.Push(wpool.Task{
		Info: TaskInfo{Action: core.MANAGE_ACTION_REDEPLOY},
		ID:   challengeName,
	})
}
