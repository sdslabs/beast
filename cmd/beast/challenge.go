package main

import (
	"os"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/core/utils"
	wpool "github.com/sdslabs/beastv4/pkg/workerpool"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var challengeCmd = &cobra.Command{
	Use:   "challenge action [challname] [-atld]",
	Short: "Performs action to the challs",
	Long:  "Performs actions like : deploy, undeploy, redeploy, purge, to the challs",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		action := args[0]

		if action == core.MANAGE_ACTION_SHOW {

			if AllChalls {
				errors := manager.ShowAllChallenges()

				if len(errors) > 0 {
					for _, err := range errors {
						log.Errorf("The following errors occurred: %s", err.Error())
					}
					os.Exit(1)
				}

			} else if Tag != "" {
				errors := manager.ShowTagRelatedChallenges(Tag)

				for _, err := range errors {
					log.Errorf("The following errors occurred: %s", err.Error())
					os.Exit(1)
				}
			} else {
				if len(args) == 1 {
					log.Errorf("Provide chall name")
					os.Exit(1)
				}

				challenge, err := database.QueryChallengeEntries("name", args[1])
				if err != nil {
					log.Errorf("Cannot query database for the given challenge : %s", err.Error())
					os.Exit(1)
				}

				if len(challenge) > 0 {
					errors := manager.ShowChallenge(challenge[0])
					for _, err := range errors {
						log.Errorf("Cannot query database for challenges ports : %s", err.Error())
						os.Exit(1)
					}

				} else {
					log.Errorf("Provide valid chall name")
					os.Exit(1)
				}

			}

		} else {

			challAction, ok := manager.ChallengeActionHandlers[action]
			if !ok {
				log.Errorf("No action %s exists", action)
				os.Exit(1)
			}

			completionChannel := make(chan bool)

			manager.Q = wpool.InitQueue(core.MAX_QUEUE_SIZE, completionChannel)
			manager.Q.StartWorkers(&manager.Worker{})

			if AllChalls {
				errstrings := manager.HandleAll(action, core.BEAST_LOCAL_SERVER)
				if len(errstrings) != 0 {
					log.Errorf("Following errors occurred : %s", strings.Join(errstrings, " || "))
					os.Exit(1)
				} else {
					log.Info("The action will be performed")
				}
			} else if Tag != "" {
				errstrings := manager.HandleTagRelatedChallenges(action, Tag, core.BEAST_LOCAL_SERVER)
				if len(errstrings) != 0 {
					log.Errorf("Following errors occurred : %s", strings.Join(errstrings, " || "))
					os.Exit(1)
				} else {
					log.Info("The action will be performed")
				}
			} else if LocalDirectory != "" {
				err := manager.DeployChallengePipeline(LocalDirectory)
				if err != nil {
					log.Errorf("Following errors occurred : %v", err)
					os.Exit(1)
				}
			} else {
				if len(args) == 1 {
					log.Errorf("Provide chall name")
					os.Exit(1)
				}
				err := challAction(args[1])
				if err != nil {
					log.Errorf("The action was not performed due to error : %s", err.Error())
					os.Exit(1)
				} else {
					log.Info("The action will be performed")
				}
			}
			_ = <-completionChannel

			if DeleteEntry && action == core.MANAGE_ACTION_PURGE {
				log.Info("Deleting database entry")
				if len(args) == 1 {
					log.Errorf("Provide chall name")
					os.Exit(1)
				}
				if err := utils.DeleteChallengeEntryWithPorts(args[1]); err != nil {
					log.Error(err)
					os.Exit(1)
				}
			}
		}
	},
}
