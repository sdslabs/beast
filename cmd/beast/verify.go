package main

import (
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Verifies the challenge config
var verifyCmd = &cobra.Command{
	Use:   "verify challenge-name",
	Short: "Verifies challenge config",
	Args:  cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		challengeName := args[0]
		var challengeStagingDir string
		for _, gitRemote := range config.Cfg.GitRemotes {
			if gitRemote.Active == true {
				challengeStagingDir = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, gitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR, challengeName)

				err := manager.ValidateChallengeConfig(challengeStagingDir)
				if err != nil {
					log.Warnf("Error while validating challenge (%s,%s) : %s", challengeName, gitRemote.RemoteName, err.Error())
				} else {
					log.Infof("The challenge config is verified.")
					break
				}
			}
		}
	},
}
