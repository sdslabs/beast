package utils

import (
	"fmt"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/database"
	"github.com/sdslabs/beastv4/docker"
	tools "github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

func ShowLogs(challname string) {
	chall, err := database.QueryFirstChallengeEntry("name", challname)
	if err != nil {
		log.Errorf("Error while database access : %s", err)
		return
	}
	if chall.Format == core.STATIC_CHALLENGE_TYPE_NAME {
		log.Info("The challenge is a static challenge")
		return
	}
	if !tools.IsContainerIdValid(chall.ContainerId) {
		log.Info("The container id is not valid")
		return
	}
	docker.ShowDockerLogsLive(chall.ContainerId)
}

func GetLogs(challname string) ([]string, error) {
	chall, err := database.QueryFirstChallengeEntry("name", challname)
	if err != nil {
		return nil, fmt.Errorf("Error while database access : %s", err)
	}
	if chall.Format == core.STATIC_CHALLENGE_TYPE_NAME {
		return nil, fmt.Errorf("The challenge is a static challenge")
	}
	if !tools.IsContainerIdValid(chall.ContainerId) {
		return nil, fmt.Errorf("The container id is not valid")
	}
	return docker.GiveDockerLogs(chall.ContainerId)
}
