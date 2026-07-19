package main

import (
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/spf13/cobra"
)

var backupDatabase = &cobra.Command{
	Use:   "backup-database",
	Short: "Backs up the configured PostgreSQL database",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cleanup, err := initializeMaintenance(false)
		if err != nil {
			return err
		}
		defer cleanup()
		return database.BackupDatabase()
	},
}

var backupCache = &cobra.Command{
	Use:   "backup-cache",
	Short: "Backs up the configured Redis database",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cleanup, err := initializeMaintenance(true)
		if err != nil {
			return err
		}
		defer cleanup()
		return cache.BackupCache()
	},
}
