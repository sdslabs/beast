package main

import (
	"github.com/sdslabs/beastv4/core/database"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var resetDatabaseCmd = &cobra.Command{
	Use:   "reset-database",
	Short: "Backups the existing database and cleans up old db and remote/staging directories",
	Run: func(cmd *cobra.Command, args []string) {
		database.BackupAndReset()
	},
}

var restoreDatabaseCmd = &cobra.Command{
	Use:   "restore-database",
	Short: "Restores the database, with the backed-up file",
	Run: func(cmd *cobra.Command, args []string) {
		if RestoreFile != "" {
			err := database.RestoreDatabase(RestoreFile)
			if err != nil {
				log.Errorf("Error restoring database from file %s: %v\n", RestoreFile, err)
			}
		} else {
			log.Fatalf("Restore file not specified.")
		}
	},
}


var backupDatabaseCmd = &cobra.Command{
	Use:   "backup-database",
	Short: "Backups the existing database and remote/staging directories",
	Run: func(cmd *cobra.Command, args []string) {
		database.BackupDatabase()
	},
}
