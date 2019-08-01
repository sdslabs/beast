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
	Long:  "Performs actions like : deploy, undeploy, redeploy, purge, show to the challs",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		action := args[0]

		if action == core.MANAGE_ACTION_SHOW {

			if AllChalls {
				challenges, err := database.QueryAllChallenges()
				if err != nil {
					log.Errorf("The action was not performed due to error : %s", err.Error())
					os.Exit(1)
				} else {
					log.Info("Name\t")
					log.Info("ContainerId\t")
					log.Info("ImageId\t")
					log.Info("Status\n")

					for _, challenge := range challenges {
						log.Info("%s\t", challenge.Name)
						log.Info("%s\t", challenge.ContainerId)
						log.Info("%s\t", challenge.ImageId)
						log.Info("%s\n", challenge.Status)
					}
				}
			} else if Tag != "" {
				// challenges, err := database.QueryRelatedChallenges(Tag)
				// if err != nil {
				// 	log.Errorf("The action was not performed due to error : %s", err.Error())
				// 	os.Exit(1)
				// } else {
				// log.Info("Name\t")
				// log.Info("ContainerId\t")
				// log.Info("ImageId\t")
				// log.Info("Status\n")

				// for _, challenge := range challenges {
				// log.Info("%s\t", challenge.Name)
				// log.Info("%s\t", challenge.ContainerId)
				// log.Info("%s\t", challenge.ImageId)
				// log.Info("%s\n", challenge.Status)
				// }
				// }
				log.Info("Tag query")
			} else {
				if len(args) == 1 {
					log.Errorf("Provide chall name")
					os.Exit(1)
				}

				challenge, err := database.QueryChallengeEntries("name", args[1])
				if err != nil {
					log.Errorf("The action was not performed due to error : %s", err.Error())
					os.Exit(1)
				}

				var challName string
				var challContainerId string
				var challImageId string
				var challStatus string

				if len(challenge) > 0 {
					challName = args[1]
					challContainerId = challenge[0].ContainerId
					challImageId = challenge[0].ImageId
					challStatus = challenge[0].Status

					log.Info("Name       :%s", challName)
					log.Info("ContainerId:%s", challContainerId)
					log.Info("ImageId    :%s", challImageId)
					log.Info("Status     :%s", challStatus)

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
