package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var healthProbeCmd = &cobra.Command{
	Use:   "health-probe",
	Short: "Run Health Probe",
	Long:  "Run Health Probe only without API server",

	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()
		database.Init()
		cache.Init()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go manager.BeastHeathCheckProber(config.Cfg.TickerFrequency)
		<-sigChan

		log.Infoln("Health probe stopped.")
	},
}
