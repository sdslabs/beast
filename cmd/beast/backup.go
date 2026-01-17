package main

import (
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/spf13/cobra"
)

var backupDatabase = &cobra.Command{
	Use:   "backup-database",
	Short: "Backups the existing database and remote/staging directories",
	Run: func(cmd *cobra.Command, args []string) {
		database.BackupDatabase()
	},
}

var backupCache = &cobra.Command{
	Use:   "backup-cache",
	Short: "Backups the existing cache and remote/staging directories",
	Run: func(cmd *cobra.Command, args []string) {
		cache.BackupCache()
	},
}
