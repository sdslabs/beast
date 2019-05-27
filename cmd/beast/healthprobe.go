package main

import (
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

var healthProbeCmd = &cobra.Command{
	Use:   "health-probe",
	Short: "Run Health Probe",
	Long:  "Run Health Probe only without API server",

	Run: func(cmd *cobra.Command, args []string) {
		utils.ChallengesHealthTicker(config.Cfg.TickerFrequency)
	},
}
