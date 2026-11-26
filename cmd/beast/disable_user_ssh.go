package main

import (
	"github.com/sdslabs/beastv4/core/auth"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/spf13/cobra"
)

var disableUserSSH = &cobra.Command{
	Use:   "disable-author-ssh",
	Short: "Disables current authors to ssh into the containers",
	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()
		database.Init()
		auth.DisableUserSSH()
	},
}
