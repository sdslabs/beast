package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// automatically generate documentation for beast cli using cobra markdown docs
var cmdRef = &cobra.Command{
	Use:   "cmdref [-r]",
	Short: "Generate beast command reference",
	Run: func(cmd *cobra.Command, args []string) {

		if RefDirectory != "" {
			err := doc.GenMarkdownTree(rootCmd, RefDirectory)
			if err != nil {
				log.Fatal(err)

			}
		} else {
			err := doc.GenMarkdownTree(rootCmd, DEFAULT_CMDREF_DIRECTORY)
			if err != nil {
				log.Fatal(err)

			}
		}
	},
}
