package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var healthProbeCmd = &cobra.Command{
	Use:   "health-probe",
	Short: "Run Health Probe",
	Long:  "Run Health Probe only without API server",

	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitConfig(); err != nil {
			log.Error(err)
			return
		}
		controllerLock, err := acquireControllerLock(core.BEAST_GLOBAL_DIR)
		if err != nil {
			log.Error(err)
			return
		}
		defer releaseControllerLock(controllerLock)

		ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stopSignals()
		manager.BeastHealthCheckProber(ctx, config.Cfg.TickerFrequency)
	},
}
