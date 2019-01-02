package cmd

import (
	"fmt"
	"os"

	"github.com/sdslabs/beastv4/cmd/debug"
	"github.com/spf13/cobra"
)

// Flag which defines verbose nature of beast
// If true beast will run in Verbose mode and will log all logs in debug
// error level.
var Verbose bool

var Port string

var KeyFile string

var Username string

var Host string

// Root command `beast` all commands are either a flag to this command
// or a subcommand for this.
var rootCmd = &cobra.Command{
	Use:   "beast",
	Short: "Beast is an automatic deployment tool for Backdoor",
	Long:  LONG_DESCRIPTION_BEAST,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if Verbose {
			debug.Enable()
		} else {
			debug.Disable()
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
	getAuthCmd.PersistentFlags().StringVarP(&KeyFile, "identity", "i", "", "Private File location")
	getAuthCmd.PersistentFlags().StringVarP(&Username, "username", "u", "", "Username")
	getAuthCmd.PersistentFlags().StringVarP(&Host, "host", "H", "http://localhost:5005/", "Hostname or IP along with port where beast is hosted")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(getAuthCmd)
}
