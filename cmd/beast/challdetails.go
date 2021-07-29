package main

import (
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

var challDetailsCmd = &cobra.Command{
	Use:   "chall-details",
	Short: "Lists all challenge details",
	Long:  "Lists all challenge details | Flags available : --status , --tags. Status flag can take arguments : deployed / undeployed / queued. Tags flag can take multiple arguments seperated with ',' : challenges-with-tags (Ex : --tags=pwn,image,docker). This displays the list of challenges which have at least one tag provided by the user. User can also use both the flags for filtering challenges",

	Run: func(cmd *cobra.Command, args []string) {
		utils.ShowChallengesInfo(cmd, args)
	},
}
