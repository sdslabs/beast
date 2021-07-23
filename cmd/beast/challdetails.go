package main

import (
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

var challdetailsCmd = &cobra.Command{
	Use:   "challdetails",
	Short: "Lists all challenge details",

	Run: func(cmd *cobra.Command, args []string) {
		utils.ShowChallengeInfo(cmd, args)
	},
}
