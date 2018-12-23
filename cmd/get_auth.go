package cmd

import (
	"github.com/sdslabs/beastv4/client"
	"github.com/spf13/cobra"
)

var getAuthCmd = &cobra.Command{
	Use:   "getauth",
	Short: "Gets Auth token from beast server",
	Long:  "Gets Auth Token from the beast server by completing the challenge from the server",

	Run: func(cmd *cobra.Command, args []string) {
		client.GetAuth(KeyFile, Host, Username)
	},
}
