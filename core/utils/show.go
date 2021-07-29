package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/utils"
	"github.com/spf13/cobra"
)

const maxShowLen int = 20

func ShowAllChallenges() []error {
	challenges, err := database.QueryAllChallenges()
	var errs []error

	if err != nil {
		errs = append(errs, err)
		return errs
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 30, 8, 2, ' ', tabwriter.Debug)
	PrintTableHeader(w)

	for _, challenge := range challenges {
		s := []string{challenge.Name, utils.TruncateString(challenge.ContainerId, maxShowLen), utils.TruncateString(challenge.ImageId, maxShowLen), challenge.Status}
		fmt.Fprint(w, strings.Join(s, "\t"))
		fmt.Fprint(w, "\t")
		ports, err := database.GetAllocatedPorts(challenge)
		if err != nil {
			errs = append(errs, err)
		} else {
			for _, port := range ports {
				fmt.Fprint(w, " ", port.PortNo)
			}
		}
		fmt.Fprintln(w)
	}
	w.Flush()

	return errs
}

func ShowTagRelatedChallenges(Tag string) []error {
	tagEntry := &database.Tag{
		TagName: Tag,
	}
	challenges, err := database.QueryRelatedChallenges(tagEntry)
	var errs []error

	if err != nil {
		errs = append(errs, err)
		return errs
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 30, 8, 2, ' ', tabwriter.Debug)
	PrintTableHeader(w)

	for _, challenge := range challenges {
		s := []string{challenge.Name, utils.TruncateString(challenge.ContainerId, maxShowLen), utils.TruncateString(challenge.ImageId, maxShowLen), challenge.Status}
		fmt.Fprint(w, strings.Join(s, "\t"))
		fmt.Fprint(w, "\t")
		ports, err := database.GetAllocatedPorts(challenge)
		if err != nil {
			errs = append(errs, err)
		} else {
			for _, port := range ports {
				fmt.Fprint(w, " ", port.PortNo)
			}
		}
		fmt.Fprintln(w)
	}
	w.Flush()

	return errs
}

func ShowChallenge(chall database.Challenge) []error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 30, 8, 2, ' ', tabwriter.Debug)
	PrintTableHeader(w)
	var errs []error

	s := []string{chall.Name, utils.TruncateString(chall.ContainerId, maxShowLen), utils.TruncateString(chall.ImageId, maxShowLen), chall.Status}
	fmt.Fprint(w, strings.Join(s, "\t"))
	fmt.Fprint(w, "\t")
	ports, err := database.GetAllocatedPorts(chall)

	if err != nil {
		errs = append(errs, err)
	} else {
		for _, port := range ports {
			fmt.Fprint(w, " ", port.PortNo)
		}
	}
	fmt.Fprintln(w)
	w.Flush()

	return errs
}

func ShowChallengeByName(name string) []error {
	challenge, err := database.QueryChallengeEntries("name", name)
	var errs []error

	if err != nil {
		errs = append(errs, err)
		return errs
	}

	if len(challenge) > 0 {
		err := ShowChallenge(challenge[0])
		for _, e := range err {
			errs = append(errs, e)
			return errs
		}
	} else {
		errs = append(errs, errors.New("Provide valid chall name"))
		return errs
	}

	return errs
}

func PrintTableHeader(w *tabwriter.Writer) {
	line := strings.Repeat("-", 180)
	fmt.Fprintln(w, "Name\tContainerId\tImageId\tStatus\tPorts")
	w.Flush()
	fmt.Println(line)
}

// ShowChallengesInfo logs information about all the challenges present in the database
func ShowChallengesInfo(cmd *cobra.Command, args []string) error {
	challenges, err := database.QueryAllChallenges()
	if err != nil {
		return fmt.Errorf("Database query error : %v", err)
	}
	var filteredChallenges []database.Challenge
	if len(challenges) > 0 {
		status, _ := cmd.Flags().GetString("status")
		status = strings.ToLower(status)
		tags, _ := cmd.Flags().GetString("tags")
		tags = strings.ToLower(tags)
		tagArray := strings.Split(tags, ",")

		for _, challenge := range challenges {
			var includeChallenge bool
			if status == "all" && tags == "all" {
				includeChallenge = true
			} else if status == "all" {
				if tagsIntersection(tagArray, &challenge) {
					includeChallenge = true
				}
			} else if tags == "all" {
				if status == strings.ToLower(challenge.Status) {
					includeChallenge = true
				}
			} else if status == strings.ToLower(challenge.Status) && tagsIntersection(tagArray, &challenge) {
				includeChallenge = true
			}

			if includeChallenge {
				filteredChallenges = append(filteredChallenges, challenge)
			}
		}
	} else {
		fmt.Errorf("No challenges present in the database.")
		return nil
	}

	if len(filteredChallenges) > 0 {
		header := []string{"Name", "Type", "Container ID", "Image ID", "Status", "Tagname", "healthcheck", "Ports"}
		border := utils.CreateBorder(true, false, true, false)
		tableConfigs := utils.CreateTableConfigs(border, header, "|")
		data := challengesData(filteredChallenges)
		utils.LogTable(tableConfigs, data)
	} else {
		fmt.Errorf("No challenges found for the provided filters.")
	}

	return nil
}

// tagsIntersection returns true if there exists an intersection between tags passed
// by the user and tags of the challenge, else returns false
func tagsIntersection(tagsInput []string, challenge *database.Challenge) bool {
	for _, tagIn := range tagsInput {
		for _, tag := range challenge.Tags {
			if strings.ToLower(tagIn) == strings.ToLower(tag.TagName) {
				return true
			}
		}
	}

	return false
}

// concatenateAllPorts returns a string of all the ports of the queried challenge
func concatenateAllPorts(ports []database.Port) string {
	var portNumber string
	for _, port := range ports {
		portNumber = portNumber + fmt.Sprint(port.PortNo) + " "
	}
	return portNumber
}

// concatenateAllTags returns a string of all the tags of the queried challenge
func concatenateAllTags(tags []*database.Tag) string {
	var tagName string
	for _, tag := range tags {
		tagName = tagName + tag.TagName + " "
	}
	return tagName
}

// challengeData returns a 2D array of data of queried challenges
func challengesData(challenges []database.Challenge) [][]string {
	data := [][]string{}
	for _, challenge := range challenges {
		var tagName string = concatenateAllTags(challenge.Tags)
		var portNumber string = concatenateAllPorts(challenge.Ports)
		row := []string{
			challenge.Name,
			challenge.Type,
			challenge.ContainerId[0:15],
			challenge.ImageId[0:15],
			challenge.Status,
			tagName,
			fmt.Sprintf("%d", challenge.HealthCheck),
			portNumber,
		}
		data = append(data, row)
	}
	return data
}
