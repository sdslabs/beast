package utils

import (
	"time"

	"github.com/sdslabs/beastv4/core/config"
)

func CheckTime() (error, int) {

	competitionInfo, err := config.GetCompetitionInfo()
	if err != nil {
		return err, -1
	}

	st, et, err := competitionInfo.ParseWindow()
	if err != nil {
		return err, -1
	}
	currentTime := time.Now().In(st.Location())

	if currentTime.Before(st) {
		return nil, 0
	}

	if currentTime.After(et) {
		return nil, 2
	}

	return nil, 1
}
