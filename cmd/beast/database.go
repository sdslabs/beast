package main

import (
	"github.com/sdslabs/beastv4/core/database"
	"github.com/spf13/cobra"
)

var resetDatabase = &cobra.Command{
	Use:   "reset-database",
	Short: "Backups the existing database and cleans up old db and remote/staging directories",
	Run: func(cmd *cobra.Command, args []string) {
		database.BackupAndReset()
	},
}
