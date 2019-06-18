package main

import (
	"github.com/sdslabs/beastv4/core/auth"
	"github.com/spf13/cobra"
)

var disableAuthorSSH = &cobra.Command{
	Use:   "disable-author-ssh",
	Short: "Disables current authors to ssh into the containers",
	Run: func(cmd *cobra.Command, args []string) {
		auth.DisableAuthorSSH()
	},
}
