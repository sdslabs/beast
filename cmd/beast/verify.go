package main

import (
	"os"

	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Verifies the challenge config
var verifyCmd = &cobra.Command{
	Use:   "verify challenge-name-or-dir",
	Short: "Verifies challenge config",
	Args:  cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()
		challengeTarget := args[0]

		challengeDir := coreUtils.GetChallengeDir(challengeTarget)
		if challengeDir == "" {
			if info, err := os.Stat(challengeTarget); err == nil && info.IsDir() {
				challengeDir = challengeTarget
			}
		}
		if challengeDir == "" {
			log.Errorf("Challenge does not exist: %s", challengeTarget)
			os.Exit(1)
		}

		err := manager.ValidateChallengeConfig(challengeDir)
		if err != nil {
			log.Errorf("Error while validating challenge %s : %s", challengeTarget, err.Error())
			os.Exit(1)
		}
		log.Infof("The challenge config is verified.")
	},
}
