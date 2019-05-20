package main

import (
	"os"

	"github.com/sdslabs/beastv4/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Beast API server",
	Long:  "Run beast API server using beast/api/server, optionally an argument can be provided to specify the port to run the server on.",

	Run: func(cmd *cobra.Command, args []string) {
		err := runBeastBootsteps()
		if err != nil {
			log.Error("Error while running Beast bootsteps.")
			os.Exit(1)
		}

		api.RunBeastApiServer(Port, !StopTicker)
	},
}
