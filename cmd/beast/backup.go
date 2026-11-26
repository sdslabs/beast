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
	Short: "Back up the PostgreSQL database",
	Run: func(cmd *cobra.Command, args []string) {
		if err := database.BackupDatabase(); err != nil {
			log.Errorf("Error backing up database: %v", err)
			os.Exit(1)
		}
	},
}

var backupCache = &cobra.Command{
	Use:   "backup-cache",
	Short: "Back up the Redis cache",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cache.BackupCache(); err != nil {
			log.Errorf("Error backing up cache: %v", err)
			os.Exit(1)
		}
	},
}
