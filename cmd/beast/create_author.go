package main

import (
	"fmt"
	"os"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
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
		if Username == "" {
			fmt.Printf("Username of Author not provided")
			os.Exit(1)
		}

		if Email == "" {
			fmt.Printf("Email not provided")
			os.Exit(1)
		}

		if PublicKeyPath == "" {
			fmt.Printf("Public Key Path not provided")
		}

		if Password == "" {
			fmt.Printf("Password not provided")
			os.Exit(1)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()

		auth.Init(core.ITERATIONS, core.HASH_LENGTH, core.TIMEPERIOD, core.ISSUER, config.Cfg.JWTSecret)

		utils.CreateAuthor(Name, Username, Email, PublicKeyPath, Password)
	},
}
