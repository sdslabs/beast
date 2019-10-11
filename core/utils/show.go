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
