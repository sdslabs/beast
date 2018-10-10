package git

import (
	"fmt"
	"path/filepath"

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
	if err != nil {
		eMsg := fmt.Errorf("Directory for the remote does not exist, clone remote first")
		log.Errorf("%s", eMsg)
		return eMsg
	}

	log.Debugf("Remote : %s, SSHKEY file : %s, Branch : %s", remote, config.Cfg.GitRemote.Secret, config.Cfg.GitRemote.Branch)
	err = pull(remote, config.Cfg.GitRemote.Secret, config.Cfg.GitRemote.Branch)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("Beast git base synced with remote")
	return nil
}

func RunBeastBootsetps() error {
	log.Info("Syncing beast git challenge dir with remote....")

	_ = SyncBeastRemote()
	return nil
}
