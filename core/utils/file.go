package utils

import (
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/utils"
)

func GetChallengeDirFromGitRemote(challengeName string) string {
	var challengeRemoteDir string

	for _, gitRemote := range config.Cfg.GitRemotes {
		if gitRemote.Active == true {
			challengeRemoteDir = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR,
				gitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR, challengeName)
			err := utils.ValidateDirExists(challengeRemoteDir)
			if err == nil {
				return challengeRemoteDir
			}
		}
	}

	return challengeRemoteDir
}
