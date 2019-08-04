package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

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
				challenges, err := database.QueryAllChallenges()
				if err != nil {
					log.Errorf("Cannot query database for challenges : %s", err.Error())
					os.Exit(1)
				} else {

					PrintTableHeader()
					w := new(tabwriter.Writer)
					var errors []error

					for _, challenge := range challenges {

						s := []string{challenge.Name, challenge.ContainerId[0:7], challenge.ImageId[0:7], challenge.Status}
						fmt.Fprint(w, strings.Join(s, "\t"))
						fmt.Fprint(w, "\t")
						ports, err := database.GetAllocatedPorts(challenge)
						if err != nil {
							errors = append(errors, err)

						}

						for _, port := range ports {
							fmt.Fprint(w, " ", port.PortNo)
						}
						fmt.Fprintln(w)
					}

					w.Flush()

					for _, err := range errors {
						log.Errorf("Cannot query database for challenges ports : %s", err.Error())
					}

				}
			} else if Tag != "" {
				tagEntry := &database.Tag{
					TagName: Tag,
				}
				challenges, err := database.QueryRelatedChallenges(tagEntry)
				if err != nil {
					log.Errorf("Cannot query database for related challenges : %s", err.Error())
					os.Exit(1)
				} else {
					PrintTableHeader()
					w := new(tabwriter.Writer)
					var errors []error

					for _, challenge := range challenges {
						s := []string{challenge.Name, challenge.ContainerId[0:7], challenge.ImageId[0:7], challenge.Status}
						fmt.Fprint(w, strings.Join(s, "\t"))
						fmt.Fprint(w, "\t")
						ports, err := database.GetAllocatedPorts(challenge)
						if err != nil {
							errors = append(errors, err)
						}

						for _, port := range ports {
							fmt.Fprint(w, " ", port.PortNo)
						}
						fmt.Fprintln(w)
					}

					w.Flush()

					for _, err := range errors {
						log.Errorf("Cannot query database for challenges ports : %s", err.Error())
					}
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
					PrintTableHeader()
					w := new(tabwriter.Writer)
					var errors []error

					s := []string{args[1], challenge[0].ContainerId[0:7], challenge[0].ImageId[0:7], challenge[0].Status}
					fmt.Fprint(w, strings.Join(s, "\t"))
					fmt.Fprint(w, "\t")
					ports, err := database.GetAllocatedPorts(challenge[0])
					if err != nil {
						errors = append(errors, err)
					}

					for _, port := range ports {
						fmt.Fprint(w, " ", port.PortNo)
					}
					fmt.Fprintln(w)
					w.Flush()

					for _, err := range errors {
						log.Errorf("Cannot query database for challenges ports : %s", err.Error())
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

func PrintTableHeader() {
	w := new(tabwriter.Writer)
	line := strings.Repeat("-", 180)
	w.Init(os.Stdout, 30, 8, 2, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Name\tContainerId\tImageId\tStatus\tPorts")
	w.Flush()
	fmt.Println(line)

}
