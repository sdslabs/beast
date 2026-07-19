package main

import (
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

var challDetailsCmd = &cobra.Command{
	Use:   "chall-details",
	Short: "Lists all challenge details",
	Long:  "Lists all challenge details | Flags available : --status , --tags. Status flag can take arguments : deployed / undeployed / queued. Tags flag can take multiple arguments seperated with ',' : (Ex : --tags=pwn,image,docker). Details are shown for challenges that have specified status and one of the specified tags.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cleanup, err := initializeCLIRuntime(false, false)
		if err != nil {
			return err
		}
		defer cleanup()
		return utils.ShowFilteredChallengesInfo(cmd, args)
	},
}
