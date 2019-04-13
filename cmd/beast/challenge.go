package main

import (
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	wpool "github.com/sdslabs/beastv4/pkg/workerpool"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var challengeCmd = &cobra.Command{
	Use:   "challenge action [challname] [-at]",
	Short: "Performs action to the challs",
	Long:  "Performs actions like : deploy, undeploy, redeploy, purge to the challs",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		challAction, ok := manager.ChallengeActionHandlers[action]
		if !ok {
			log.Errorf("No action %s exists", action)
			return
		}

		completionChannel := make(chan bool)

		manager.Q = wpool.InitQueue(core.MAX_QUEUE_SIZE, completionChannel)
		manager.Q.StartWorkers(&manager.Worker{})

		if AllChalls {
			errstrings := manager.HandleAll(action, core.BEAST_LOCAL_SERVER)
			if len(errstrings) != 0 {
				log.Errorf("Following errors occurred : %s", strings.Join(errstrings, " || "))
				return
			} else {
				log.Info("The action will be performed")
			}
		} else if Tag != "" {
			errstrings := manager.HandleTagRelatedChallenges(action, Tag, core.BEAST_LOCAL_SERVER)
			if len(errstrings) != 0 {
				log.Errorf("Following errors occurred : %s", strings.Join(errstrings, " || "))
				return
			} else {
				log.Info("The action will be performed")
			}
		} else {
			if len(args) == 1 {
				log.Errorf("Provide chall name")
				return
			}
			err := challAction(args[1])
			if err != nil {
				log.Errorf("The action was not performed due to error : %s", err.Error())
				return
			} else {
				log.Info("The action will be performed")
			}
		}
		_ = <-config.Channels
	},
}
