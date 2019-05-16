package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func runBeastBootsteps() error {
	log.Debug("Running Beast bootsteps.")
	return nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Run Beast initial setup bootsetps.",
	Long:  "Initializes beast by setting up beast directory, checking for permission. It also configures the logger and local SQLite database to be used by beast",

	Run: func(cmd *cobra.Command, args []string) {
		runBeastBootsteps()
	},
}
