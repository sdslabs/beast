package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Root command `beast` all commands are either a flag to this command
// or a subcommand for this.
var rootCmd = &cobra.Command{
	Use:   "beast",
	Short: "Beast is an automatic deployment tool for Backdoor",
	Long:  LONG_DESCRIPTION_BEAST,

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
func init() {
	rootCmd.AddCommand(versionCmd)
}
