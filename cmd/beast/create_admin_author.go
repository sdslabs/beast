package main

import (
	"fmt"
	"os"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/spf13/cobra"
)

func createAuthorAdminPrereq() {
	config.InitConfig()
	database.Init()

	auth.Init(core.ITERATIONS, core.HASH_LENGTH, core.TIMEPERIOD, core.ISSUER, config.Cfg.JWTSecret, []string{core.USER_ROLES["author"]}, []string{core.USER_ROLES["admin"]}, []string{core.USER_ROLES["contestant"]})
}

var createAuthorCmd = &cobra.Command{
	Use:   "create-author",
	Short: "Creates new author",
	Long:  "Creates new author using command line arguments",
	PreRun: func(cmd *cobra.Command, args []string) {
		if Name == "" {
			fmt.Println("Name of Author not provided")
			os.Exit(1)
		}
		if Username == "" {
			fmt.Println("Username of Author not provided")
			os.Exit(1)
		}

		if Email == "" {
			fmt.Println("Email not provided")
			os.Exit(1)
		}

		if PublicKeyPath == "" {
			fmt.Println("Public Key Path not provided")
		}

		if Password == "" {
			fmt.Println("Password not provided")
			os.Exit(1)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		createAuthorAdminPrereq()
		if err := utils.CreateAdminOrAuthor(Name, Username, Email, PublicKeyPath, Password, "author"); err != nil {
			os.Exit(1)
		}
	},
}

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Creates new admin",
	Long:  "Creates new admin using command line arguments",
	PreRun: func(cmd *cobra.Command, args []string) {
		if Name == "" {
			fmt.Println("Name of Admin not provided")
			os.Exit(1)
		}
		if Username == "" {
			fmt.Println("Username of Admin not provided")
			os.Exit(1)
		}

		if Email == "" {
			fmt.Println("Email not provided")
			os.Exit(1)
		}

		if PublicKeyPath == "" {
			fmt.Println("Public Key Path not provided")
		}

		if Password == "" {
			fmt.Println("Password not provided")
			os.Exit(1)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		createAuthorAdminPrereq()
		if err := utils.CreateAdminOrAuthor(Name, Username, Email, PublicKeyPath, Password, "admin"); err != nil {
			os.Exit(1)
		}
	},
}
