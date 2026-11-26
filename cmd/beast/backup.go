package main

import (
	"os"

	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/database"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var backupDatabase = &cobra.Command{
	Use:   "backup-database",
	Short: "Backups the existing database and remote/staging directories",
	Run: func(cmd *cobra.Command, args []string) {
		if err := database.BackupDatabase(); err != nil {
			log.Errorf("Error backing up database: %v", err)
			os.Exit(1)
		}
	},
}

var backupCache = &cobra.Command{
	Use:   "backup-cache",
	Short: "Backups the existing cache and remote/staging directories",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cache.BackupCache(); err != nil {
			log.Errorf("Error backing up cache: %v", err)
			os.Exit(1)
		}
	},
}
