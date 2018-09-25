package main

import (
	"fmt"
	"os"

	"github.com/fristonio/beast/cmd"
	"github.com/fristonio/beast/core"
	_ "github.com/fristonio/beast/core/database"
	log "github.com/sirupsen/logrus"
)

func init() {
	// Check if the beast directory exist, if it does not exist then create it
	// if an error occurs in between exit the utility printing the error.
	if _, err := os.Stat(core.BEAST_GLOBAL_DIR); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(core.BEAST_GLOBAL_DIR, 0755); err != nil {
				fmt.Println("Error occured while creatind beast dir")
				os.Exit(1)
			}
		} else {
			fmt.Println("Error while checking beast dir stats, check permissions")
			os.Exit(1)
		}
	}

	// Setup logger for the application.
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true

	log.SetFormatter(Formatter)
	log.SetLevel(log.WarnLevel)

	log.Debug("Setting up logging complete for beast")
}

func main() {
	cmd.Execute()
}
