package main

import (
	"github.com/sdslabs/beastv4/core/manager"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
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
		challengeStagingDir = coreUtils.GetChallengeDirFromGitRemote(challengeName)

		err := manager.ValidateChallengeConfig(challengeStagingDir)
		if err != nil {
			log.Warnf("Error while validating challenge %s : %s", challengeName, err.Error())
		} else {
			log.Infof("The challenge config is verified.")
		}
	},
}
