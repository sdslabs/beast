package api

import (
	"github.com/sdslabs/beastv4/core/config"
)

func getLogoPath() string {
	competitionInfo, err := config.GetCompetitionInfo()
	if err != nil {
		return "404"
	}

	if competitionInfo.LogoURL == "" {
		return "404"
	}

	return competitionInfo.LogoURL
}
