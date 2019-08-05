package main

import (
	"github.com/spf13/cobra/doc"
	// "path/filepath"
	// "github.com/sdslabs/beastv4/core"
	// "github.com/sdslabs/beastv4/core/config"
	// "github.com/sdslabs/beastv4/core/manager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// automatically generate documentation for beast cli using cobra markdown docs
var cmdRef = &cobra.Command{
	Use:   "cmdref [-d]",
	Short: "Generate beast command reference",
	Run: func(cmd *cobra.Command, args []string) {

		if RefDirectory != "" {
			genCustomCmdRef()
		} else {
			genCmdRef()
		}
	},
}

func genCmdRef() {

	err := doc.GenMarkdownTree(rootCmd, "docs/cmdref/")
	if err != nil {
		log.Fatal(err)

	}
}

func genCustomCmdRef() {
	err := doc.GenMarkdownTree(rootCmd, RefDirectory)
	if err != nil {
		log.Fatal(err)

	}
}
