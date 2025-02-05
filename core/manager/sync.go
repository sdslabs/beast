package manager

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/pkg/git"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

// Sync the beast remote directory with the actual git repository.
func SyncBeastRemote(defaultauthorpassword string) error {
	log.Info("Syncing local challenge repository with remote.")
	beastRemoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	dirGitRemote := make(map[string]string)
	var errStrings []string
	for _, gitRemote := range config.Cfg.GitRemotes {
		if gitRemote.Active {
			remote := filepath.Join(beastRemoteDir, gitRemote.RemoteName)

			err := utils.ValidateDirExists(remote)
			log.Debugf("Remote : %s, SSHKEY file : %s, Branch : %s", remote, gitRemote.Secret, gitRemote.Branch)

			if err != nil {
				log.Warnf("Directory for the remote(%s) does not exist", remote)
				log.Infof("Performing initial repository clone, this may take a while...")

				err = git.Clone(remote, gitRemote.Secret, gitRemote.Url, gitRemote.Branch, core.GIT_DEFAULT_REMOTE)
				if err != nil {
					log.Errorf("Error while cloning repository : %s", err)
					errors := fmt.Errorf("error while cloning repository : %s", err)
					errStrings = append(errStrings, errors.Error())
					continue
				}
			}

			log.Debugf("Pulling latest changes from the remote.")

			err = utils.ValidateFileExists(gitRemote.Secret)
			if err != nil {
				errors := fmt.Errorf("error while validating file location : %s : %v", gitRemote.Secret, err)
				errStrings = append(errStrings, errors.Error())
				continue
			}
			err = git.Pull(remote, gitRemote.Secret, gitRemote.Branch, core.GIT_DEFAULT_REMOTE)
			if err != nil {
				if !strings.Contains(err.Error(), "already up-to-date") {
					log.Errorf("Error while syncing beast with git remote : %s ...", err)
					errStrings = append(errStrings, err.Error())
					continue
				} else {
					log.Infof("GIT remote already synced")
				}
			}
			if dirGitRemote[remote] == "" {
				dirGitRemote[remote] = gitRemote.RemoteName
			} else {
				err := fmt.Errorf("directory exist in multiple git repository")
				errStrings = append(errStrings, err.Error())
				continue
			}
		}
	}
	log.Info("Beast git base synced with remote")
	go config.UpdateUsedPortList()
	UpdateChallenges(defaultauthorpassword)
	return fmt.Errorf("%s", strings.Join(errStrings, "\n"))
}

func ResetBeastRemote(defaultauthorpassword string) error {
	var errStrings []string
	log.Debugf("Cleaning existing remote directories")
	for _, gitRemote := range config.Cfg.GitRemotes {
		if gitRemote.Active {
			remoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, gitRemote.RemoteName)
			err := utils.RemoveDirRecursively(remoteDir)
			if err != nil {
				e := fmt.Errorf("error while cloning repository : %s", err)
				log.Error(e)
				errStrings = append(errStrings, e.Error())
			}
		}
	}
	err := SyncBeastRemote(defaultauthorpassword)
	if err != nil {
		log.Errorf("Error while syncing remote after clean : %s", err)
	}
	errors := strings.Join(errStrings, "\n") + err.Error()
	return fmt.Errorf("%s", errors)
}

// IsAlreadySynced checks if the local repository is already synced
func IsAlreadySynced() bool {
	for _, gitRemote := range config.Cfg.GitRemotes {
		if gitRemote.Active {
			gitDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, gitRemote.RemoteName)
			err := utils.ValidateDirExists(gitDir)
			if err != nil {
				return false
			}

			synced, err := git.IsAlreadyUpToDate(gitDir, gitRemote.Secret, gitRemote.Branch, core.GIT_DEFAULT_REMOTE)
			if err != nil {
				log.Errorf("Error while checking if local repo already synced: %s", err)
				return false
			}
			if !synced {
				return false
			}
		}
	}

	log.Info("Local challenge repository already synced")
	return true
}

// SyncAndGetChangesFromRemote gets changes from remote since the last sync
// Returns an array of names of challenges which were modified
func SyncAndGetChangesFromRemote(defaultauthorpassword string) []string {
	log.Info("Syncing local challenge repository with remote.")
	var modifiedChallsNameList []string

	alreadySynced := IsAlreadySynced()
	if alreadySynced {
		return modifiedChallsNameList
	}

	beastRemoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	gitRemoteDirSet := utils.EmptySet()
	for _, gitRemote := range config.Cfg.GitRemotes {
		if gitRemote.Active {
			remote := filepath.Join(beastRemoteDir, gitRemote.RemoteName)
			if !gitRemoteDirSet.Contains(remote) {
				gitRemoteDirSet.Add(remote)
			} else {
				log.Errorf("Directory already in use for another remote")
				continue
			}

			err := utils.ValidateDirExists(remote)
			log.Debugf("Remote: %s, Branch: %s", remote, gitRemote.Branch)

			if err != nil {
				log.Warnf("Directory for the remote(%s) does not exist", remote)
				log.Infof("Performing initial repository clone, this may take a while...")

				err = git.Clone(remote, gitRemote.Secret, gitRemote.Url, gitRemote.Branch, core.GIT_DEFAULT_REMOTE)
				if err != nil {
					log.Errorf("Error while cloning repository: %s", err)
					continue
				}

				challengesDirRoot := filepath.Join(remote, core.BEAST_REMOTE_CHALLENGE_DIR)
				err, challenges := utils.GetDirsInDir(challengesDirRoot)
				if err != nil {
					log.Errorf("Error while getting available challenges for the remote(%s): %s", remote, err)
					continue
				}

				modifiedChallsNameList = append(modifiedChallsNameList, challenges...)
			}

			log.Debugf("Pulling latest changes from the remote.")

			err = utils.ValidateFileExists(gitRemote.Secret)
			if err != nil {
				log.Errorf("Error while validating file location: %s: %s", gitRemote.Secret, err)
				continue
			}

			filesChanged, err := git.PullAndGetChanges(remote, gitRemote.Secret, gitRemote.Branch, core.GIT_DEFAULT_REMOTE)
			if err != nil {
				if !strings.Contains(err.Error(), "already up-to-date") {
					log.Errorf("Error while syncing beast with git remote: %s", err)
					continue
				}
				log.Infof("GIT remote already synced")
			}

			challenges := ExtractChallengeNamesFromFileNames(filesChanged)
			modifiedChallsNameList = append(modifiedChallsNameList, challenges...)
		}
	}
	log.Info("Beast git base synced with remote")
	go config.UpdateUsedPortList()
	UpdateChallenges(defaultauthorpassword)

	return modifiedChallsNameList
}

func RunBeastBootsteps(defaultauthorpassword string) error {
	log.Info("Syncing beast git challenge dir with remote....")

	_ = SyncBeastRemote(defaultauthorpassword)
	return nil
}
