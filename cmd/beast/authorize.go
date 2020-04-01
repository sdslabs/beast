package main

import (
	"fmt"
	"os"

	"github.com/sdslabs/beastv4/client"
	"github.com/spf13/cobra"
)

var getAuthCmd = &cobra.Command{
	Use:   "getauth",
	Short: "Gets Auth token from beast server",
	Long:  "Gets Auth Token from the beast server by completing the challenge from the server",
	PreRun: func(cmd *cobra.Command, args []string) {
		if Password == "" {
			fmt.Printf("Password not provided")
			os.Exit(1)
		}

		if Username == "" {
			fmt.Printf("Username not provided")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		client.Authorize(Password, Host, Username)
	},
}
