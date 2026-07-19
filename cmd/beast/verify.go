package main

import (
	"fmt"

	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

// Verifies the challenge config
var verifyCmd = &cobra.Command{
	Use:   "verify challenge-name",
	Short: "Verifies challenge config",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.InitConfig(); err != nil {
			return err
		}
		challengeName := args[0]

		challengeDir := coreUtils.GetChallengeDir(challengeName)
		if challengeDir == "" {
			return fmt.Errorf("challenge %q does not exist", challengeName)
		}

		if err := manager.ValidateChallengeConfig(challengeDir); err != nil {
			return fmt.Errorf("validate challenge %s: %w", challengeName, err)
		}
		return nil
	},
}
