package git

import (
	"path/filepath"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

// Sync the beast remote directory with the actual git repository.
func SyncBeastRemote() error {
	beastRemoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	remote := filepath.Join(beastRemoteDir, config.Cfg.GitRemote.RemoteName)

	err := utils.ValidateDirExists(remote)
	log.Debugf("Remote : %s, SSHKEY file : %s, Branch : %s", remote, config.Cfg.GitRemote.Secret, config.Cfg.GitRemote.Branch)

	if err != nil {
		log.Warnf("Directory for the remote(%s) does not exist", remote)
		log.Infof("Performing initial repository clone, this may take a while...")

		err = clone(remote, config.Cfg.GitRemote.Secret, config.Cfg.GitRemote.Url, config.Cfg.GitRemote.Branch)
		if err != nil {
			log.Errorf("Error while cloning repository : %s", err)
			return err
		}

		return nil
	}

	log.Debugf("Pulling latest changes from the remote.")
	err = pull(remote, config.Cfg.GitRemote.Secret, config.Cfg.GitRemote.Branch)
	if err != nil {
		if !strings.Contains(err.Error(), "already up-to-date") {
			log.Errorf("Error while syncing beast with git remote : %s ...", err)
			return err
		} else {
			log.Infof("GIT remote already synced")
		}
	}

	log.Info("Beast git base synced with remote")
	go config.UpdateUsedPortList()
	return nil
}

func ResetBeastRemote() error {
	remoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, config.Cfg.GitRemote.RemoteName)

	log.Debugf("Cleaning existing remote directory")
	err := utils.RemoveDirRecursively(remoteDir)
	if err != nil {
		log.Error(err)
		return err
	}

	err = SyncBeastRemote()
	if err != nil {
		log.Errorf("Error while syncing remote after clean : %s", err)
		return err
	}

	return nil
}

func RunBeastBootsetps() error {
	log.Info("Syncing beast git challenge dir with remote....")

	_ = SyncBeastRemote()
	return nil
}
