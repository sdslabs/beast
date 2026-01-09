package utils

import (
	"fmt"
	"time"
)

var hours = []string{
	"00", "01", "02", "03", "04", "05", "06", "07", "08", "09",
	"10", "11", "12", "13", "14", "15", "16", "17", "18", "19",
	"20", "21", "22", "23",
}

var minutes = []string{
	"00", "01", "02", "03", "04", "05", "06", "07", "08", "09",
	"10", "11", "12", "13", "14", "15", "16", "17", "18", "19",
	"20", "21", "22", "23", "24", "25", "26", "27", "28", "29",
	"30", "31", "32", "33", "34", "35", "36", "37", "38", "39",
	"40", "41", "42", "43", "44", "45", "46", "47", "48", "49",
	"50", "51", "52", "53", "54", "55", "56", "57", "58", "59",
}

var months = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

var timezones = []string{
	"UTC",
	"Local",
}

func parseMonth(input string) time.Month {
	t, err := time.Parse("January", input)
	if err != nil {
		return time.January
	}

	return t.Month()
}

func daysIn(input string, year int) int {
	month := parseMonth(input)
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func getTime(year int, monthName string, day int, hour int, minute int, useUTC bool) time.Time {
	month := parseMonth(monthName)

	var loc *time.Location
	if useUTC {
		loc = time.UTC
	} else {
		loc = time.Local
	}

	return time.Date(year, month, day, hour, minute, 0, 0, loc)
}

func FormatTime(dateTime time.Time) string {
	_, offset := dateTime.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hrs := offset / 3600
	mins := (offset % 3600) / 60

	day := dateTime.Day()
	suffix := "th"
	if day%10 == 1 && day != 11 {
		suffix = "st"
	} else if day%10 == 2 && day != 12 {
		suffix = "nd"
	} else if day%10 == 3 && day != 13 {
		suffix = "rd"
	}

	formatted := fmt.Sprintf("%02d:%02d:%02d UTC: %s%02d:%02d, %d%s %s %d, %s",
		dateTime.Hour(), dateTime.Minute(), dateTime.Second(),
		sign, hrs, mins,
		day, suffix, dateTime.Month(), dateTime.Year(), dateTime.Weekday())

	return formatted
}
