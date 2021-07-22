package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/alexeyco/simpletable"
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

func ShowChallengeInfo(cmd *cobra.Command, args []string) error {

	challenges, err := database.QueryAllChallenges()
	Status, _ := cmd.Flags().GetString("status")
	Status = strings.ToLower(Status)

	if err != nil {
		return fmt.Errorf("DATABASE ERROR while processing the request :%v", err)
	}

	if len(challenges) > 0 {

		var portNumber string
		for _, challenge := range challenges {
			for _, port := range challenge.Ports {
				portNumber = portNumber + fmt.Sprint(port.PortNo) + " "
			}
		}

		var tagName string
		for _, challenge := range challenges {
			for _, tag := range challenge.Tags {
				tagName = tagName + fmt.Sprint(tag.TagName) + " "
			}
		}

		table := simpletable.New()

		table.Header = &simpletable.Header{
			Cells: []*simpletable.Cell{
				{Align: simpletable.AlignCenter, Text: "Chall Name"},
				{Align: simpletable.AlignCenter, Text: "Chall Type"},
				{Align: simpletable.AlignCenter, Text: "Chall Container ID"},
				{Align: simpletable.AlignCenter, Text: "Chall Image ID"},
				{Align: simpletable.AlignCenter, Text: "Chall Status"},
				{Align: simpletable.AlignCenter, Text: "Chall Tagname"},
				{Align: simpletable.AlignCenter, Text: "Chall healthcheck"},
				{Align: simpletable.AlignCenter, Text: "Chall Ports"},
			},
		}

		for _, challenge := range challenges {

			if Status == "deployed" && challenge.Status != "Deployed" {
				continue
			}
			if Status == "undeployed" && challenge.Status != "Undeployed" {
				continue
			}
			if Status == "queued" && challenge.Status != "Queued" {
				continue
			}

			r := []*simpletable.Cell{
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("%s", challenge.Name)},
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("%s", challenge.Type)},
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("%s", challenge.ContainerId[0:15])},
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("%s", challenge.ImageId[0:15])},
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("%s", challenge.Status)},
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("%s", tagName)},
				{Align: simpletable.AlignCenter, Text: fmt.Sprintf("%d", challenge.HealthCheck)},
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("%s", portNumber)},
			}

			table.Body.Cells = append(table.Body.Cells, r)

		}

		table.SetStyle(simpletable.StyleCompactLite)
		fmt.Println(table.String())

	} else {
		fmt.Println("No challenges present.")
	}

	return nil
}
