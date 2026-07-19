package main

import (
	"fmt"

	"github.com/sdslabs/beastv4/core/cache"
	"github.com/spf13/cobra"
)

var resetCacheCmd = &cobra.Command{
	Use:   "reset-cache",
	Short: "Backs up and resets the configured Redis database",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDestructiveConfirmation(); err != nil {
			return err
		}
		cleanup, err := initializeMaintenance(true)
		if err != nil {
			return err
		}
		defer cleanup()
		return cache.BackupAndReset()
	},
}

var restoreCacheCmd = &cobra.Command{
	Use:   "restore-cache",
	Short: "Restores the configured Redis database from a backup",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDestructiveConfirmation(); err != nil {
			return err
		}
		if RestoreFile == "" {
			return fmt.Errorf("restore file is required")
		}
		cleanup, err := initializeMaintenance(true)
		if err != nil {
			return err
		}
		defer cleanup()
		return cache.RestoreCache(RestoreFile)
	},
}
