package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	container_types "github.com/docker/docker/api/types"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
)

func GetLogs(challname string, live bool) (*cr.Log, error) {
	chall, err := database.QueryFirstChallengeEntry("name", challname)
	if err != nil {
		return nil, fmt.Errorf("Error while database access : %s", err)
	}

	if chall.Format == core.STATIC_CHALLENGE_TYPE_NAME {
		return nil, fmt.Errorf("The challenge is a static challenge, no log present")
	}

	if !IsContainerIdValid(chall.ContainerId) {
		return nil, fmt.Errorf("Underlying challenge configuration present is not valid.")
	}
	var containers, remoteContainers []container_types.Container
	server := config.AvailableServer{}
	remoteContainers, err = remoteManager.SearchContainerByFilterRemote(map[string]string{"id": chall.ContainerId}, server)
	if err != nil {
		return nil, fmt.Errorf("Error while searching for remote container with id %s", chall.ContainerId)
	}
	containers, err = cr.SearchContainerByFilter(map[string]string{"id": chall.ContainerId})
	if err != nil {
		return nil, fmt.Errorf("Error while searching for container with id %s", chall.ContainerId)
	}
	containers = append(containers, remoteContainers...)
	if len(containers) > 1 {
		return nil, errors.New("Got more than one containers, something fishy here. Contact admin to check manually.")
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("Underlying container for getting log is not present.")
	}

	if live {
		if chall.ServerDeployed != core.LOCALHOST && chall.ServerDeployed != "" {
			server := config.Cfg.AvailableServers[chall.ServerDeployed]
			remoteManager.ShowLiveContainerLogsRemote(chall.ContainerId, server)
		} else {
			cr.ShowLiveContainerLogs(chall.ContainerId)
		}
		return nil, nil
	}
	if chall.ServerDeployed != core.LOCALHOST && chall.ServerDeployed != "" {
		server := config.Cfg.AvailableServers[chall.ServerDeployed]
		return remoteManager.GetContainerStdLogsRemote(chall.ContainerId, server)
	}
	return cr.GetContainerStdLogs(chall.ContainerId)
}

func LogCheating(msg string) error {
	// log the cheating attempt in a file in cheat.log
	file, err := os.OpenFile(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CHEAT_LOG_FILE), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error while opening cheat log file : %s", err)
	}
	defer file.Close()

	if _, err := file.WriteString(msg + "\n"); err != nil {
		return fmt.Errorf("error while appending to cheat log file : %s", err)
	}
	return nil
}
