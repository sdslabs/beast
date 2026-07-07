package main

import (
	"github.com/sdslabs/beastv4/core/cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var resetCacheCmd = &cobra.Command{
	Use:   "reset-cache",
	Short: "Backups the existing cache and cleans up old cache and remote/staging directories",
	Run: func(cmd *cobra.Command, args []string) {
		cache.BackupAndReset()
	},
}

var restoreCacheCmd = &cobra.Command{
	Use:   "restore-cache",
	Short: "Restores the cache, with the backed-up file",
	Run: func(cmd *cobra.Command, args []string) {
		if RestoreFile != "" {
			err := cache.RestoreCache(RestoreFile)
			if err != nil {
				log.Errorf("Error restoring cache from file %s: %v\n", RestoreFile, err)
			}
		} else {
			log.Fatalf("Restore file not specified.")
		}
	},
}
