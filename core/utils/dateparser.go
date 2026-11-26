package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/sdslabs/beastv4/core/config"

	"github.com/araddon/dateparse"
)

func parseBeastDateTime(raw string, loc *time.Location) (time.Time, error) {
	parts := strings.Split(raw, ",")
	if len(parts) < 2 {
		return dateparse.ParseIn(strings.TrimSpace(raw), loc)
	}
	dateFields := strings.Fields(parts[1])
	timeFields := strings.Fields(parts[0])
	if len(dateFields) < 3 || len(timeFields) < 1 {
		return time.Time{}, fmt.Errorf("invalid competition time format: %q", raw)
	}
	formatted := fmt.Sprintf("%s %s, %s, %s", dateFields[1], dateFields[0], dateFields[2], timeFields[0])
	return dateparse.ParseIn(formatted, loc)
}

func CheckTime() (error, int) {
	competitionInfo, err := config.GetCompetitionInfo()
	if err != nil {
		return err, -1
	}

	tzName := strings.TrimSpace(strings.Split(competitionInfo.TimeZone, ":")[0])
	loc, locErr := time.LoadLocation(tzName)
	if locErr != nil || loc == nil {
		loc = time.UTC
	}
	currentTime := time.Now().In(loc)

	st, err := parseBeastDateTime(competitionInfo.StartingTime, loc)
	if err != nil {
		return err, -1
	}

	et, err := parseBeastDateTime(competitionInfo.EndingTime, loc)
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
