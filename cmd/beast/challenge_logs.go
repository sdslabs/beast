package main

import (
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs CHALLNAME",
	Short: "Provides live logs of a container",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		config.InitConfig()
		database.Init()

		_, err := utils.GetLogs(args[0], true)
		return err
	},
}
