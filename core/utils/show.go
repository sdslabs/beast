package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/utils"
	"github.com/spf13/cobra"
)

const maxShowLen int = 20

func formatChallengePorts(chall database.Challenge) (string, error) {
	ports, err := database.GetAllocatedPorts(chall)
	if err != nil {
		return "", err
	}
	if len(ports) > 0 {
		parts := make([]string, 0, len(ports))
		for _, port := range ports {
			parts = append(parts, fmt.Sprint(port.PortNo))
		}
		return strings.Join(parts, " "), nil
	}

	if len(chall.Ports) > 0 {
		parts := make([]string, 0, len(chall.Ports))
		for _, port := range chall.Ports {
			parts = append(parts, fmt.Sprint(port.PortNo))
		}
		return strings.Join(parts, " "), nil
	}

	if cache.Cache != nil && chall.ServerDeployed != "" && chall.ContainerId != "" {
		redisPorts, redisErr := cache.GetContainerPortsOnHost(chall.ServerDeployed, chall.ContainerId)
		if redisErr == nil && len(redisPorts) > 0 {
			parts := make([]string, 0, len(redisPorts))
			for _, port := range redisPorts {
				parts = append(parts, fmt.Sprint(port))
			}
			return strings.Join(parts, " "), nil
		}
	}

	if chall.Instanced && cache.Cache != nil {
		instances, instErr := cache.GetChallengeInstances(chall.Name)
		if instErr == nil && len(instances) > 0 {
			var parts []string
			for _, inst := range instances {
				for _, p := range inst.Port {
					parts = append(parts, fmt.Sprint(p))
				}
			}
			if len(parts) > 0 {
				return fmt.Sprintf("instanced (%s)", strings.Join(parts, ",")), nil
			}
		}
		return "instanced", nil
	}

	return "", nil
}

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
		portStr, portErr := formatChallengePorts(challenge)
		if portErr != nil {
			errs = append(errs, portErr)
		} else if portStr != "" {
			fmt.Fprint(w, " ", portStr)
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
		portStr, portErr := formatChallengePorts(challenge)
		if portErr != nil {
			errs = append(errs, portErr)
		} else if portStr != "" {
			fmt.Fprint(w, " ", portStr)
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
	portStr, err := formatChallengePorts(chall)

	if err != nil {
		errs = append(errs, err)
	} else if portStr != "" {
		fmt.Fprint(w, " ", portStr)
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
		errs = append(errs, ShowChallenge(challenge[0])...)
	} else {
		errs = append(errs, errors.New("Provide valid chall name"))
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
	status, _ := cmd.Flags().GetString("status")
	status = strings.ToLower(strings.TrimSpace(status))
	if _, valid := core.DEPLOY_STATUS[status]; status != "all" && !valid {
		return fmt.Errorf("invalid deployment status %q", status)
	}
	tags, _ := cmd.Flags().GetString("tags")

	challenges, err := database.QueryAllChallenges()
	if err != nil {
		return fmt.Errorf("Database query error : %v", err)
	}

	filteredChallenges := filterChallenges(challenges, status, utils.SplitDelimited(tags, ","))
	if len(filteredChallenges) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No challenges match the selected filters.")
		return nil
	}

	header := []string{"Name", "Type", "Container ID", "Image ID", "Status", "Tagname", "Health Check", "Ports"}
	border := utils.CreateBorder(true, false, true, false)
	tConfigs := utils.CreateTableConfigs(border, header, "|")
	tData := getTableDataFromChallenges(filteredChallenges)
	utils.LogTable(tConfigs, tData)

	return nil
}

// tagsIntersection returns true if there exists an intersection between tags passed
// by the user and tags of the challenge, else returns false
func tagsIntersection(tagsInput []string, tags []*database.Tag) bool {
	for _, tagIn := range tagsInput {
		for _, tag := range tags {
			if strings.EqualFold(tagIn, tag.TagName) {
				return true
			}
		}
	}

	return false
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
		portNumber, err := formatChallengePorts(challenge)
		if err != nil {
			portNumber = ""
		}
		row := []string{
			challenge.Name,
			challenge.Type,
			utils.TruncateString(challenge.ContainerId, maxShowLen),
			utils.TruncateString(challenge.ImageId, maxShowLen),
			challenge.Status,
			tagName,
			fmt.Sprintf("%d", challenge.HealthCheck),
			portNumber,
		}
		data = append(data, row)
	}
	return data
}
