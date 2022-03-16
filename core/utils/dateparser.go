package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/sdslabs/beastv4/core/config"

	"github.com/araddon/dateparse"
)

func CheckTime() (error, int) {

	competitionInfo, err := config.GetCompetitionInfo()
	if err != nil {
		return err, -1
	}

	loc, _ := time.LoadLocation(strings.Split(competitionInfo.TimeZone, ":")[0])
	time.Local = loc
	currentTime := time.Now().In(loc)

	compStartTime := strings.Split(competitionInfo.StartingTime, ",")
	compStartDate := strings.Split(compStartTime[1][1:], " ")
	startDate := fmt.Sprintf("%s %s, %s", compStartDate[1], compStartDate[0], compStartDate[2])
	startTime := strings.Split(compStartTime[0], " ")[0]
	startTime = fmt.Sprintf("%s, %s", startDate, startTime)

	st, err := dateparse.ParseLocal(startTime)
	if err != nil {
		return err, -1
	}

	compEndTime := strings.Split(competitionInfo.EndingTime, ",")
	compEndDate := strings.Split(compEndTime[1][1:], " ")
	endDate := fmt.Sprintf("%s %s, %s", compEndDate[1], compEndDate[0], compEndDate[2])
	endTime := strings.Split(compEndTime[0], " ")[0]
	endTime = fmt.Sprintf("%s, %s", endDate, endTime)

	et, err := dateparse.ParseLocal(endTime)
	if err != nil {
		return err, -1
	}

	if currentTime.Before(st) {
		return nil, 0
	}

	if currentTime.After(et) {
		return nil, 2
	}

	return nil, 1
}
