package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	_ "github.com/sdslabs/beastv4/core/database"

	log "github.com/sirupsen/logrus"
)

func initDirectory(dir string) {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(dir, 0755); err != nil {
				fmt.Println("Error occured while creating beast dir")
				os.Exit(1)
			}
		} else {
			fmt.Println("Error while checking beast dir stats, check permissions")
			os.Exit(1)
		}
	}
}

func init() {
	// Check if the beast directory exist, if it does not exist then create it
	// if an error occurs in between exit the utility printing the error.
	initDirectory(core.BEAST_GLOBAL_DIR)
	initDirectory(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR))
	initDirectory(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR))

	// Setup logger for the application.
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true

	log.SetFormatter(Formatter)
	log.SetLevel(log.WarnLevel)

	log.Debug("Setting up logging complete for beast")
}

func main() {
	Execute()
}
