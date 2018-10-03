package cmd

import (
	"fmt"
	"os"

	"github.com/sdslabs/beastv4/version"
	"github.com/spf13/cobra"
)

// Print the version of beast running and exit gracefully
// Version information are retrived from the version/version.go which is populated at
// build time using ldflags
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Displays the version of the current build of beast",
	Long: `Displays the version of the current build of beast, this information
include Version, Revision, Git-Branch, BuildUser, BuildDate, go-version`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(version.VersionStr,
			version.Info["version"],
			version.Info["revision"],
			version.Info["branch"],
			version.Info["buildUser"],
			version.Info["buildDate"],
			version.Info["goVersion"])
		os.Exit(0)
	},
}
