package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
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
	Long:  "Performs actions like : deploy, undeploy, redeploy, purge to the challs",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config.InitConfig()
		database.Init()
		cache.Init()

		// Since action is already verfied to exist it does not make sense to check
		// its existence here therefore we directly parse the action from the command.
		action := args[0]
		noCache, _ := cmd.Flags().GetBool("no-cache")
		if noCache && (action == core.MANAGE_ACTION_DEPLOY || action == core.MANAGE_ACTION_REDEPLOY) {
			config.NoCache = noCache
		} else if noCache {
			log.Errorf("no-cache flag is available only for \"deploy\" and \"redeploy\" actions")
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

			return nil
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

			return manager.StartDeployPipeline(LocalDirectory, false, false, config.NoCache)
		}

		completionChannel := make(chan bool, 1)
		var errorMu sync.Mutex
		var workerErr error

		manager.Q = wpool.InitQueue(core.MAX_QUEUE_SIZE, completionChannel)
		manager.Q.StartWorkers(&manager.Worker{OnError: func(err error) {
			errorMu.Lock()
			defer errorMu.Unlock()
			workerErr = errors.Join(workerErr, err)
		}})

		var requestErr error
		if AllChalls {
			errstrings := manager.HandleAll(action, core.BEAST_LOCAL_SERVER)
			if len(errstrings) > 0 {
				requestErr = errors.New(strings.Join(errstrings, " || "))
			}
		} else if Tag != "" {
			errstrings := manager.HandleTagRelatedChallenges(action, Tag, core.BEAST_LOCAL_SERVER)
			if len(errstrings) > 0 {
				requestErr = errors.New(strings.Join(errstrings, " || "))
			}
		} else {
			if len(args) == 1 {
				return errors.New("provide a challenge name")
			}
			requestErr = challAction(args[1])
		}
		for manager.Q.Pending() > 0 {
			<-completionChannel
		}
		if err := errors.Join(requestErr, workerErr); err != nil {
			return fmt.Errorf("challenge %s: %w", action, err)
		}
		return nil
	},
}
