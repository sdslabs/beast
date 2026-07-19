package main

import (
	"fmt"

	"github.com/sdslabs/beastv4/core/database"
	"github.com/spf13/cobra"
)

var resetDatabaseCmd = &cobra.Command{
	Use:   "reset-database",
	Short: "Backs up and resets the configured PostgreSQL database",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDestructiveConfirmation(); err != nil {
			return err
		}
		cleanup, err := initializeMaintenance(false)
		if err != nil {
			return err
		}
		defer cleanup()
		return database.BackupAndReset()
	},
}

var restoreDatabaseCmd = &cobra.Command{
	Use:   "restore-database",
	Short: "Restores the configured PostgreSQL database from a backup",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDestructiveConfirmation(); err != nil {
			return err
		}
		if RestoreFile == "" {
			return fmt.Errorf("restore file is required")
		}
		cleanup, err := initializeMaintenance(false)
		if err != nil {
			return err
		}
		defer cleanup()
		return database.RestoreDatabase(RestoreFile)
	},
}
