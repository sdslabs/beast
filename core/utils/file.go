package utils

import (
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/utils"
)

func GetChallengeDir(challengeName string) string {
	if config.Cfg == nil || !config.IsValidChallengeName(challengeName) {
		return ""
	}

	for _, gitRemote := range config.Cfg.GitRemotes {
		if gitRemote.Active {
			challengeRemoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR,
				gitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR, challengeName)
			if err := utils.ValidateDirExists(challengeRemoteDir); err == nil {
				return challengeRemoteDir
			}
		}
	}

	challengeUploadDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR, challengeName)
	if err := utils.ValidateDirExists(challengeUploadDir); err == nil {
		return challengeUploadDir
	}

	return ""
}
