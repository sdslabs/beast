package main

import (
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs CHALLNAME",
	Short: "Provides live logs of a container",
	Args:  cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitConfig(); err != nil {
			log.Error(err)
			return
		}

		utils.GetLogs(args[0], true)
	},
}
