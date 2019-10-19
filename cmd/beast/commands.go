package main

import (
	"fmt"
	"os"

	"github.com/sdslabs/beastv4/cmd/debug"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/spf13/cobra"
)

var (
	Verbose           bool
	HealthProbe       bool
	Port              string
	KeyFile           string
	Username          string
	Host              string
	Name              string
	Email             string
	PublicKeyPath     string
	SkipAuthorization bool
	AllChalls         bool
	PeriodicSync      bool
	Tag               string
	LocalDirectory    string
	DeleteEntry       bool
	RefDirectory      string
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
	runCmd.PersistentFlags().BoolVarP(&HealthProbe, "health-probe", "k", false, "Run health check service for beast deployed challenges")
	runCmd.PersistentFlags().BoolVarP(&PeriodicSync, "periodic-sync", "s", false, "Periodically sync remote with beast.")
	runCmd.PersistentFlags().BoolVarP(&SkipAuthorization, "noauth", "n", false, "Skip Authorization")

	getAuthCmd.PersistentFlags().StringVarP(&KeyFile, "identity", "i", "", "Private File location")
	getAuthCmd.PersistentFlags().StringVarP(&Username, "username", "u", "", "Username")
	getAuthCmd.PersistentFlags().StringVarP(&Host, "host", "H", "http://localhost:5005/", "Hostname or IP along with port where beast is hosted")

	createAuthorCmd.PersistentFlags().StringVarP(&Name, "name", "", "", "Name of the new author")
	createAuthorCmd.PersistentFlags().StringVarP(&Email, "email", "", "", "Email of the new author")
	createAuthorCmd.PersistentFlags().StringVarP(&PublicKeyPath, "publickey", "", "", "Public key file representing new author")

	challengeCmd.PersistentFlags().BoolVarP(&AllChalls, "all", "a", false, "Performs action to all challs")
	challengeCmd.PersistentFlags().StringVarP(&Tag, "tag", "t", "", "Performs action to the tag provided")
	challengeCmd.PersistentFlags().StringVarP(&LocalDirectory, "local-directory", "l", "", "Deploys challenge from local directory")
	challengeCmd.PersistentFlags().BoolVarP(&DeleteEntry, "delete-entry", "d", false, "Deletes db entry related to this challenge")

	cmdRef.PersistentFlags().StringVarP(&RefDirectory, "reference-directory", "r", "", "Generate beast command reference files in reference directory")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(getAuthCmd)
	rootCmd.AddCommand(createAuthorCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(healthProbeCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(challengeCmd)
	rootCmd.AddCommand(disableAuthorSSH)
	rootCmd.AddCommand(cmdRef)
	rootCmd.AddCommand(generateTemplateCmd)
}
