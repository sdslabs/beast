package main

import (
	"fmt"
	"os"

	"github.com/sdslabs/beastv4/cmd/debug"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/spf13/cobra"
)

var (
	Verbose               bool
	HealthProbe           bool
	Port                  string
	DefaultAuthorPassword string
	Name                  string
	Host                  string
	Username              string
	Email                 string
	Password              string
	PublicKeyPath         string
	CsvFile			  	  string
	SkipAuthorization     bool
	AllChalls             bool
	AutoDeploy            bool
	PeriodicSync          bool
	Tag                   string
	LocalDirectory        string
	DeleteEntry           bool
	RefDirectory          string
	Status                string
	Tags                  string
	NoCache               bool
)

// Root command `beast` all commands are either a flag to this command
// or a subcommand for this.
var rootCmd = &cobra.Command{
	Use:   "beast",
	Short: "Beast is an deployment and management tool for CTF challenges.",
	Long:  LONG_DESCRIPTION_BEAST,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if Verbose {
			debug.Enable()
		} else {
			debug.Disable()
		}

		if SkipAuthorization {
			config.SkipAuthorization = true
		} else {
			config.SkipAuthorization = false
		}

		config.NoCache = NoCache
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// This function executes the root command for the tool.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Init function to add commands to the root command `beast`. It has the
// following subcommands:
// * versionCmd: A command to show version information for current build.
// * Initialize Beast, run bootsteps and check if it is configured properly
// * runCmd: Run beast API server.
func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Print extra information in stdout1")

	runCmd.PersistentFlags().StringVarP(&Port, "port", "p", "", "Port to run the beast server on.")
	runCmd.PersistentFlags().StringVarP(&DefaultAuthorPassword, "defaultauthorpassword", "q", "", "Default password for creating author, users are not created if value is empty string")
	runCmd.PersistentFlags().BoolVarP(&AutoDeploy, "auto-deploy", "a", false, "Auto deploy all challenges from remote on server start.")
	runCmd.PersistentFlags().BoolVarP(&HealthProbe, "health-probe", "k", false, "Run health check service for beast deployed challenges")
	runCmd.PersistentFlags().BoolVarP(&PeriodicSync, "periodic-sync", "s", false, "Periodically sync remote with beast and auto update challenges.")
	runCmd.PersistentFlags().BoolVarP(&SkipAuthorization, "noauth", "n", false, "Skip Authorization")
	runCmd.PersistentFlags().BoolVarP(&NoCache, "no-cache", "c", false, "Build image of challenge without using cache")

	getAuthCmd.PersistentFlags().StringVarP(&Username, "username", "u", "", "Username")
	getAuthCmd.PersistentFlags().StringVarP(&Password, "password", "p", "", "Password")
	getAuthCmd.PersistentFlags().StringVarP(&Host, "host", "H", "http://localhost:5005/", "Hostname or IP along with port where beast is hosted")

	createAuthorCmd.PersistentFlags().StringVarP(&Name, "name", "", "", "Name of the new author")
	createAuthorCmd.PersistentFlags().StringVarP(&Username, "username", "", "", "Username of the new author")
	createAuthorCmd.PersistentFlags().StringVarP(&Password, "password", "", "", "Password of the author")
	createAuthorCmd.PersistentFlags().StringVarP(&Email, "email", "", "", "Email of the new author")
	createAuthorCmd.PersistentFlags().StringVarP(&PublicKeyPath, "publickey", "", "", "Public key file representing new author")

	createMultipleAuthorCmd.PersistentFlags().StringVarP(&CsvFile, "csv", "", "", "CSV file containing details of author")

	createAdminCmd.PersistentFlags().StringVarP(&Name, "name", "", "", "Name of the new admin")
	createAdminCmd.PersistentFlags().StringVarP(&Username, "username", "", "", "Username of the new admin")
	createAdminCmd.PersistentFlags().StringVarP(&Password, "password", "", "", "Password of the admin")
	createAdminCmd.PersistentFlags().StringVarP(&Email, "email", "", "", "Email of the new admin")
	createAdminCmd.PersistentFlags().StringVarP(&PublicKeyPath, "publickey", "", "", "Public key file representing new admin")

	createMultipleAdminCmd.PersistentFlags().StringVarP(&CsvFile, "csv", "", "", "CSV file containing details of author")

	challengeCmd.PersistentFlags().BoolVarP(&AllChalls, "all", "a", false, "Performs action to all challs")
	challengeCmd.PersistentFlags().StringVarP(&Tag, "tag", "t", "", "Performs action to the tag provided")
	challengeCmd.PersistentFlags().StringVarP(&LocalDirectory, "local-directory", "l", "", "Deploys challenge from local directory")
	challengeCmd.PersistentFlags().BoolVarP(&DeleteEntry, "delete-entry", "d", false, "Deletes db entry related to this challenge")
	challengeCmd.PersistentFlags().BoolVarP(&NoCache, "no-cache", "c", false, "Build image of challenge without using cache")

	cmdRef.PersistentFlags().StringVarP(&RefDirectory, "reference-directory", "r", "", "Generate beast command reference files in reference directory")

	challDetailsCmd.PersistentFlags().StringVarP(&Status, "status", "s", "all", "Filter by status : deployed / undeployed / queued")
	challDetailsCmd.PersistentFlags().StringVarP(&Tags, "tags", "t", "", "Filter by tagname : pwn / web / image / docker")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(getAuthCmd)
	rootCmd.AddCommand(createAuthorCmd)
	rootCmd.AddCommand(createAdminCmd)
	rootCmd.AddCommand(createMultipleAuthorCmd)
	rootCmd.AddCommand(createMultipleAdminCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(healthProbeCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(challengeCmd)
	rootCmd.AddCommand(disableUserSSH)
	rootCmd.AddCommand(cmdRef)
	rootCmd.AddCommand(generateTemplateCmd)
	rootCmd.AddCommand(challDetailsCmd)
}
