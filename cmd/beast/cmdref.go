package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// automatically generate documentation for beast cli using cobra markdown docs
var cmdRef = &cobra.Command{
	Use:   "cmdref [-r]",
	Short: "Generate beast command reference",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		directory := RefDirectory
		if directory == "" {
			directory = DEFAULT_CMDREF_DIRECTORY
		}
		if err := os.MkdirAll(directory, 0750); err != nil {
			return fmt.Errorf("create command reference directory: %w", err)
		}
		return doc.GenMarkdownTree(rootCmd, directory)
	},
}
