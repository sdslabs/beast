package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

var binaryOptions = []string{
	"yes",
	"no",
}

func PromptString(prompt string) string {
	log.Println(prompt)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		log.Errorln("Failed to read input... defaulting to empty string...")
		return ""
	}

	return scanner.Text()
}

func PromptSecret(prompt string) string {
	log.Println(prompt)
	bytes, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		log.Errorln("Failed to read input... defaulting to empty string...")
	}

	return string(bytes)
}

func PromptInt64(prompt string, defaultValue int64) int64 {
	log.Println(fmt.Sprintf("%s (defaults to %v)", prompt, defaultValue))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		log.Errorln(fmt.Sprintf("Failed to read input... defaulting to %v...", defaultValue))
		return defaultValue
	}

	temp := scanner.Text()
	tempInt, err := strconv.ParseInt(temp, 10, 64)

	if temp == "" {
		log.Warnln(fmt.Sprintf("Input empty.. defaulting to %v...", defaultValue))
		return defaultValue
	} else if err != nil {
		log.Errorln(fmt.Sprintf("Failed to read input... defaulting to %v...", defaultValue))
		return defaultValue
	}

	return tempInt
}

func PromptFloat32(prompt string, defaultValue float32) float32 {
	log.Println(fmt.Sprintf("%s (defaults to %v)", prompt, defaultValue))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		log.Errorln(fmt.Sprintf("Failed to read input... defaulting to %v...", defaultValue))
		return defaultValue
	}

	temp := scanner.Text()
	tempFloat, err := strconv.ParseFloat(temp, 32)

	if temp == "" {
		log.Warnln(fmt.Sprintf("Input empty.. defaulting to %v...", defaultValue))
		return defaultValue
	} else if err != nil {
		log.Errorln(fmt.Sprintf("Failed to read input... defaulting to %v...", defaultValue))
		return defaultValue
	}

	return float32(tempFloat)
}

func PromptSelection(prompt string, items []string) string {
	selection := promptui.Select{
		Label: prompt,
		Items: items,
	}

	_, result, err := selection.Run()
	if err != nil {
		log.Errorln(fmt.Sprintf("Failed to read input... defaulting to %s...", items[0]))
		return items[0]
	}

	return result
}

func PromptBinary(prompt string) bool {
	selection := promptui.Select{
		Label: prompt,
		Items: binaryOptions,
	}

	_, result, err := selection.Run()
	if err != nil {
		log.Errorln("Failed to read input... defaulting to no...")
		return false
	}

	return result == "yes"
}

func PromptDateTime(prompt string) time.Time {
	log.Println(prompt)

	now := time.Now()
	year := now.Year()

	yearSelection := promptui.Select{
		Label: "Enter Year",
		Items: []int{
			year,
			year + 1,
			year + 2,
		},
	}

	_, selectedYear, err := yearSelection.Run()
	if err != nil {
		log.Errorln("Failed to read input... defaulting to now...")
		return now
	}

	monthSelection := promptui.Select{
		Label: "Enter Month",
		Items: months,
		Size:  5,
	}

	_, selectedMonth, err := monthSelection.Run()
	if err != nil {
		log.Errorln("Failed to read input... defaulting to now...")
		return now
	}

	parsedYear, _ := strconv.Atoi(selectedYear)

	numDays := daysIn(selectedMonth, parsedYear)
	days := make([]int, numDays)
	for i := 0; i < numDays; i++ {
		days[i] = i + 1
	}

	daySelection := promptui.Select{
		Label: "Enter Day",
		Items: days,
		Size:  5,
	}

	_, selectedDay, err := daySelection.Run()
	if err != nil {
		log.Errorln("Failed to read input... defaulting to now...")
		return now
	}

	hourSelection := promptui.Select{
		Label: "Enter Hour",
		Items: hours,
		Size:  5,
	}

	_, selectedHour, err := hourSelection.Run()
	if err != nil {
		log.Errorln("Failed to read input... defaulting to now...")
		return now
	}

	minuteSelection := promptui.Select{
		Label: "Enter Minute",
		Items: minutes,
		Size:  5,
	}

	_, selectedMinute, err := minuteSelection.Run()
	if err != nil {
		log.Errorln("Failed to read input... defaulting to now...")
		return now
	}

	timezone := PromptSelection("Select timezone", timezones)

	parsedDay, _ := strconv.Atoi(selectedDay)
	parsedHour, _ := strconv.Atoi(selectedHour)
	parsedMinute, _ := strconv.Atoi(selectedMinute)

	return getTime(parsedYear, selectedMonth, parsedDay, parsedHour, parsedMinute, timezone == timezones[0])
}
