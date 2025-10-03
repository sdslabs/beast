package main

import (
	"errors"
	"github.com/sdslabs/beastv4/core"
	"os"
	"os/signal"
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
		if _, err := os.Stat(core.BEAST_GLOBAL_DIR); errors.Is(err, os.ErrNotExist) {
			log.Infoln(".beast directory not found... running Beast bootsteps...")

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
