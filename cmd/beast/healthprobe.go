package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/spf13/cobra"
)

var healthProbeCmd = &cobra.Command{
	Use:   "health-probe",
	Short: "Run Health Probe",
	Long:  "Run Health Probe only without API server",

	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cleanup, err := initializeCLIRuntime(true, true)
		if err != nil {
			return err
		}
		defer cleanup()
		controllerLock, err := acquireControllerLock(core.BEAST_GLOBAL_DIR)
		if err != nil {
			return err
		}
		defer releaseControllerLock(controllerLock)

		ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stopSignals()
		manager.BeastHealthCheckProber(ctx, config.Cfg.TickerFrequency)
		return nil
	},
}
