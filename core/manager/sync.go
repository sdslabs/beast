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
func SyncBeastRemote() error {
	log.Info("Syncing local challenge repository with remote.")
	beastRemoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	dirGitRemote := make(map[string]string)
	var errStrings []string
	for _, gitRemote := range config.Cfg.GitRemotes {
		if gitRemote.Active == true {
			remote := filepath.Join(beastRemoteDir, gitRemote.RemoteName)

			err := utils.ValidateDirExists(remote)
			log.Debugf("Remote : %s, SSHKEY file : %s, Branch : %s", remote, gitRemote.Secret, gitRemote.Branch)

			if err != nil {
				log.Warnf("Directory for the remote(%s) does not exist", remote)
				log.Infof("Performing initial repository clone, this may take a while...")

				err = git.Clone(remote, gitRemote.Secret, gitRemote.Url, gitRemote.Branch, core.GIT_DEFAULT_REMOTE)
				if err != nil {
					log.Errorf("Error while cloning repository : %s", err)
					errors := fmt.Errorf("Error while cloning repository : %s", err)
					errStrings = append(errStrings, errors.Error())
					continue
				}
			}

			log.Debugf("Pulling latest changes from the remote.")

			err = utils.ValidateFileExists(gitRemote.Secret)
			if err != nil {
				errors := fmt.Errorf("Error while validating file location : %s : %v", gitRemote.Secret, err)
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
				err := fmt.Errorf("Directory exist in multiple git repository")
				errStrings = append(errStrings, err.Error())
				continue
			}
		}
	}
	log.Info("Beast git base synced with remote")
	go config.UpdateUsedPortList()
	return fmt.Errorf(strings.Join(errStrings, "\n"))
}

func ResetBeastRemote() error {
	var errStrings []string
	log.Debugf("Cleaning existing remote directories")
	for _, gitRemote := range config.Cfg.GitRemotes {
		if gitRemote.Active == true {
			remoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, gitRemote.RemoteName)
			err := utils.RemoveDirRecursively(remoteDir)
			if err != nil {
				e := fmt.Errorf("Error while cloning repository : %s", err)
				log.Error(e)
				errStrings = append(errStrings, e.Error())
			}
		}
	}
	err := SyncBeastRemote()
	if err != nil {
		log.Errorf("Error while syncing remote after clean : %s", err)
	}
	errors := strings.Join(errStrings, "\n") + err.Error()
	return fmt.Errorf(errors)
}

func RunBeastBootsetps() error {
	log.Info("Syncing beast git challenge dir with remote....")

	_ = SyncBeastRemote()
	return nil
}
