package utils

import (
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/utils"
)

func GetChallengeDir(challengeName string) string {
	challengeRemoteDir := ""

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

	challengeRemoteDir = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR, challengeName)
	err := utils.ValidateDirExists(challengeRemoteDir)
	if err == nil {
		return challengeRemoteDir
	}

	return challengeRemoteDir
}
