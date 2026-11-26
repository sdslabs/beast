package main

import (
	"os"

	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var challDetailsCmd = &cobra.Command{
	Use:   "chall-details",
	Short: "Lists all challenge details",
	Long:  "Lists all challenge details | Flags available : --status , --tags. Status flag can take arguments : deployed / undeployed / queued. Tags flag can take multiple arguments seperated with ',' : (Ex : --tags=pwn,image,docker). Details are shown for challenges that have specified status and one of the specified tags.",

	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()
		database.Init()
		cache.Init()
		if err := utils.ShowFilteredChallengesInfo(cmd, args); err != nil {
			log.Errorf("Error showing challenge details: %v", err)
			os.Exit(1)
		}
	},
}
