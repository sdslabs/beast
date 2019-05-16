package main

import (
	"fmt"
	"os"

	"github.com/sdslabs/beastv4/core/utils"
	"github.com/spf13/cobra"
)

var createAuthorCmd = &cobra.Command{
	Use:   "create-author",
	Short: "Creates new author",
	Long:  "Creates new author using command line arguments",
	PreRun: func(cmd *cobra.Command, args []string) {
		if Name == "" {
			fmt.Printf("Name of Author not provided")
			os.Exit(1)
		}

		if Email == "" {
			fmt.Printf("Email not provided")
			os.Exit(1)
		}

		if PublicKeyPath == "" {
			fmt.Printf("Public Key Path not provided")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		utils.CreateAuthor(Name, Email, PublicKeyPath)
	},
}
