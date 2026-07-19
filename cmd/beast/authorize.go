package main

import (
	"fmt"
	"strings"

	"github.com/sdslabs/beastv4/client"
	"github.com/sdslabs/beastv4/utils"
	"github.com/spf13/cobra"
)

var getAuthCmd = &cobra.Command{
	Use:   "getauth",
	Short: "Gets an authentication token from the Beast server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(Username) == "" {
			return fmt.Errorf("username is required")
		}
		password := utils.PromptSecret("Enter Beast password")
		if password == "" {
			return fmt.Errorf("password is required")
		}
		response, err := client.Authorize(password, Host, Username, AuthCAFile)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Token\t: %s\nMessage\t: %s\n", response.Token, response.Message)
		return nil
	},
}
