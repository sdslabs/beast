package main

import (
	"errors"

	"github.com/sdslabs/beastv4/client"
	"github.com/spf13/cobra"
)

var getAuthCmd = &cobra.Command{
	Use:   "getauth",
	Short: "Gets an auth token from the Beast server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if Password == "" || Username == "" {
			return errors.New("username and password are required")
		}
		return client.Authorize(Password, Host, Username)
	},
}
