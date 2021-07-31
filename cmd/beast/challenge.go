package main

import (
	"os"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/core/utils"
	wpool "github.com/sdslabs/beastv4/pkg/workerpool"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var challengeCmd = &cobra.Command{
	Use:   "challenge action [challname] [-atld]",
	Short: "Performs action to the challs",
	Long:  "Performs actions like : deploy, undeploy, redeploy, purge to the challs",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()

		// Since action is already verfied to exist it does not make sense to check
		// its existence here therefore we directly parse the action from the command.
		action := args[0]
		noCache, _ := cmd.Flags().GetBool("no-cache")
		if noCache && action == core.MANAGE_ACTION_DEPLOY {
			config.NoCache = noCache
		} else if noCache {
			log.Errorf("no-cache flag is available only for \"deploy\" action")
			os.Exit(1)
		}

		if action == core.MANAGE_ACTION_SHOW {

			if AllChalls {
				errors := utils.ShowAllChallenges()

				if len(errors) > 0 {
					for _, err := range errors {
						log.Errorf("The following errors occurred: %s", err.Error())
					}
					os.Exit(1)
				}

			} else if Tag != "" {
				errors := utils.ShowTagRelatedChallenges(Tag)

				if len(errors) > 0 {
					for _, err := range errors {
						log.Errorf("The following errors occurred: %s", err.Error())
					}
					os.Exit(1)
				}
			} else {
				if len(args) == 1 {
					log.Errorf("Provide chall name")
					os.Exit(1)
				}

				errors := utils.ShowChallengeByName(args[1])
				if len(errors) > 0 {
					for _, err := range errors {
						log.Errorf("The following errors occurred: %s", err.Error())
					}
					os.Exit(1)
				}

			}

			return
		}

		challAction, ok := manager.ChallengeActionHandlers[action]
		if !ok {
			log.Errorf("No action %s exists", action)
			os.Exit(1)
		}

		// Handle local directory deployment separately.
		if LocalDirectory != "" {
			if action != core.MANAGE_ACTION_DEPLOY {
				log.Errorf("Only deploy action is available for the challenge with local directory")
				os.Exit(1)
			}

			manager.StartDeployPipeline(LocalDirectory, false, false, false)
			return
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
	},
}
