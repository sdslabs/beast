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

// ShowFilteredChallengesInfo logs info of challenges filtered by provided status and tags
func ShowFilteredChallengesInfo(cmd *cobra.Command, args []string) error {

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
		var tagsArray []string
		if len(tags) > 0 {
			tagsArray = strings.Split(tags, ",")
		}
		filteredChallenges = filterChallenges(challenges, status, tagsArray)
	} else {
		fmt.Errorf("No challenges present in the database.")
		return nil
	}

	if len(filteredChallenges) > 0 {
		header := []string{"Name", "Type", "Container ID", "Image ID", "Status", "Tagname", "Health Check", "Ports"}
		border := utils.CreateBorder(true, false, true, false)
		tConfigs := utils.CreateTableConfigs(border, header, "|")
		tData := getTableDataFromChallenges(filteredChallenges)
		utils.LogTable(tConfigs, tData)
	} else {
		fmt.Errorf("No challenges found for the provided filters.")
	}

	return nil
}

// tagsIntersection returns true if there exists an intersection between tags passed
// by the user and tags of the challenge, else returns false
func tagsIntersection(tagsInput []string, tags []*database.Tag) bool {
	for _, tagIn := range tagsInput {
		for _, tag := range tags {
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

// filterChallenges returns an array of challenges filtered by status and tags
func filterChallenges(challenges []database.Challenge, status string, tags []string) []database.Challenge {
	var filteredChallenges []database.Challenge

	for _, challenge := range challenges {
		var includeChallenge bool

		if status == "all" && len(tags) == 0 {
			includeChallenge = true
		} else if status == "all" {
			includeChallenge = tagsIntersection(tags, challenge.Tags)
		} else if len(tags) == 0 {
			includeChallenge = status == strings.ToLower(challenge.Status)
		} else if status == strings.ToLower(challenge.Status) && tagsIntersection(tags, challenge.Tags) {
			includeChallenge = true
		}

		if includeChallenge {
			filteredChallenges = append(filteredChallenges, challenge)
		}
	}

	return filteredChallenges
}

// getTableDataFromChallenges returns data of provided challenges in tablewriter compatible format
func getTableDataFromChallenges(challenges []database.Challenge) [][]string {
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
