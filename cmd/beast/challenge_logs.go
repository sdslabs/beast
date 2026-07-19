package main

import (
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs CHALLNAME",
	Short: "Provides live logs of a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cleanup, err := initializeCLIRuntime(false, false)
		if err != nil {
			return err
		}
		defer cleanup()
		_, err = utils.GetLogs(args[0], true)
		return err
	},
}
