package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/sdslabs/beastv4/core/config"
)

func CheckTime() (error, int) {
	layout := "15:04:05 2 January 2006 MST"

	competitionInfo, err := config.GetCompetitionInfo()
	if err != nil {
		return err, -1
	}

	loc, _ := time.LoadLocation(strings.Split(competitionInfo.TimeZone, ":")[0])
	currentTime := time.Now().In(loc)
	timezone := strings.Split(currentTime.String(), " ")

	compStartTime := strings.Split(competitionInfo.StartingTime, ",")
	startTimeStr := fmt.Sprintf("%s%s %s", strings.Split(compStartTime[0], " ")[0], compStartTime[1], timezone[3])
	startTime, err := time.Parse(layout, startTimeStr)
	if err != nil {
		return err, -1
	}

	compEndTime := strings.Split(competitionInfo.EndingTime, ",")
	endTimeStr := fmt.Sprintf("%s%s %s", strings.Split(compEndTime[0], " ")[0], compEndTime[1], timezone[3])
	endTime, err := time.Parse(layout, endTimeStr)
	if err != nil {
		return err, -1
	}

	if currentTime.Before(startTime) {
		return nil, 0
	}

	if currentTime.After(endTime) {
		return nil, 2
	}

	return nil, 1
}
