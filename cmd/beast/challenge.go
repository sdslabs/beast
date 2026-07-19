package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sdslabs/beastv4/core"
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
		action := args[0]
		if NoCache && action != core.MANAGE_ACTION_DEPLOY {
			return fmt.Errorf("no-cache flag is available only for deploy")
		}

		if action == core.MANAGE_ACTION_SHOW {
			cleanup, err := initializeCLIRuntime(false, false)
			if err != nil {
				return err
			}
			defer cleanup()
			return showChallenges(args)
		}

		challAction, ok := manager.ChallengeActionHandlers[action]
		if !ok {
			return fmt.Errorf("no challenge action %q exists", action)
		}
		cleanup, err := initializeCLIRuntime(true, true)
		if err != nil {
			return err
		}
		defer cleanup()

		if LocalDirectory != "" {
			if action != core.MANAGE_ACTION_DEPLOY {
				return fmt.Errorf("local-directory is available only for deploy")
			}
			return manager.StartDeployPipeline(LocalDirectory, false, false, NoCache)
		}

		completion := make(chan bool, 1)
		manager.Q = wpool.InitQueue(core.MAX_QUEUE_SIZE, completion)
		manager.Q.StartWorkers(&manager.Worker{})
		defer manager.Q.Stop()

		if AllChalls {
			if failures := manager.HandleAll(action, core.BEAST_LOCAL_SERVER); len(failures) != 0 {
				return fmt.Errorf("challenge actions failed: %s", strings.Join(failures, " || "))
			}
		} else if Tag != "" {
			if failures := manager.HandleTagRelatedChallenges(action, Tag, core.BEAST_LOCAL_SERVER); len(failures) != 0 {
				return fmt.Errorf("challenge actions failed: %s", strings.Join(failures, " || "))
			}
		} else {
			if len(args) == 1 {
				return fmt.Errorf("challenge name is required")
			}
			if err := challAction(args[1]); err != nil {
				return fmt.Errorf("perform %s on %s: %w", action, args[1], err)
			}
		}

		<-completion
		if failures := manager.Q.Errors(); len(failures) != 0 {
			return errors.Join(failures...)
		}
		log.Info("Challenge action completed")
		return nil
	},
}

func showChallenges(args []string) error {
	var failures []error
	switch {
	case AllChalls:
		failures = utils.ShowAllChallenges()
	case Tag != "":
		failures = utils.ShowTagRelatedChallenges(Tag)
	case len(args) < 2:
		return fmt.Errorf("challenge name is required")
	default:
		failures = utils.ShowChallengeByName(args[1])
	}
	if len(failures) == 0 {
		return nil
	}
	messages := make([]string, 0, len(failures))
	for _, err := range failures {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf("show challenges: %s", strings.Join(messages, "; "))
}
