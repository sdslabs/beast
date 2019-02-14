package cmd

import (
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

var dockerLogsCmd = &cobra.Command{
	Use:   "docker-logs CHALLNAME",
	Short: "Provides live logs of a container",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		utils.ShowLogs(args[0])
	},
}
