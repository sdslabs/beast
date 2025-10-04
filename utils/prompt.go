package utils

import (
	"bufio"
	"fmt"
	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
	"os"
	"strconv"
	"syscall"
)

var binaryOptions = []string{
	"yes",
	"no",
}

func PromptString(prompt string) string {
	log.Infoln(prompt)

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
	log.Infoln(fmt.Sprintf("%s (defaults to %v)", prompt, defaultValue))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		log.Errorln(fmt.Sprintf("Failed to read input... defaulting to %v...", defaultValue))
		return defaultValue
	}

	temp := scanner.Text()
	tempInt, err := strconv.ParseInt(temp, 10, 64)
	if err != nil {
		log.Errorln(fmt.Sprintf("Failed to read input... defaulting to %v...", defaultValue))
		return defaultValue
	}

	return tempInt
}

func PromptSelection(prompt string, items []string) string {
	log.Println(prompt)
	selection := promptui.Select{
		Label: fmt.Sprintf("%s", prompt),
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
	log.Println(prompt)
	selection := promptui.Select{
		Label: fmt.Sprintf("%s", prompt),
		Items: binaryOptions,
	}

	_, result, err := selection.Run()
	if err != nil {
		log.Errorln("Failed to read input... defaulting to no...")
		return false
	}

	return result == "yes"
}
