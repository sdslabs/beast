package utils

import (
	"fmt"
	"os"
	"path/filepath"

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

	if chall.Type == core.STATIC_CHALLENGE_TYPE_NAME || chall.Format == core.STATIC_CHALLENGE_TYPE_NAME {
		return nil, fmt.Errorf("static challenge %q does not have container logs", challname)
	}
	if !IsContainerIdValid(chall.ContainerId) {
		return nil, fmt.Errorf("challenge %q does not have a deployed container", challname)
	}

	if config.Cfg.UseLocalDockerDaemon(chall.ServerDeployed) {
		if live {
			return nil, cr.ShowLiveContainerLogs(chall.ContainerId)
		}
		return cr.GetContainerStdLogs(chall.ContainerId)
	}

	server := config.Cfg.AvailableServers[chall.ServerDeployed]
	if live {
		return nil, remoteManager.ShowLiveContainerLogsRemote(chall.ContainerId, server)
	}
	return remoteManager.GetContainerStdLogsRemote(chall.ContainerId, server)
}

func LogFlag(msg string, challName string) error {
	// log the cheating attempt in a file in cheat.log
	file, err := os.OpenFile(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challName, core.BEAST_CHALLENGE_LOGS_DIR, core.BEAST_FLAG_LOG_FILE), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error while opening cheat log file : %s", err)
	}
	defer file.Close()

	if _, err := file.WriteString(msg + "\n"); err != nil {
		return fmt.Errorf("error while appending to flag log file : %s", err)
	}
	return nil
}
