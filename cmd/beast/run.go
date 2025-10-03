package main

import (
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sdslabs/beastv4/api"
	"github.com/sdslabs/beastv4/core/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Beast API server",
	Long:  "Run beast API server using beast/api/server, optionally an argument can be provided to specify the port to run the server on.",

	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			log.WithField("error", err.Error()).Errorf("Error trying to get home directory")
			os.Exit(1)
		}

		if _, err := os.Stat(filepath.Join(home, ".beast")); errors.Is(err, os.ErrNotExist) {
			log.Infoln(".beast directory not found... running Beast bootsteps")

			if err := runBeastBootsteps(); err != nil {
				log.Error("Error while running Beast bootsteps.")
				os.Exit(1)
			}
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go api.RunBeastApiServer(Port, DefaultAuthorPassword, AutoDeploy, HealthProbe, PeriodicSync, NoCache)
		<-sigChan

		log.Infoln("\nShutdown signal received.")
		utils.Cleanup()
		log.Infoln("Server stopped gracefully.")
	},
}
