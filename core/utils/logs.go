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
	var containers []container_types.Container
	if config.Cfg.UseLocalDockerDaemon(chall.ServerDeployed) {
		containers, err = cr.SearchContainerByFilter(map[string]string{"id": chall.ContainerId})
	} else {
		server, ok := config.Cfg.AvailableServers[chall.ServerDeployed]
		if !ok {
			return nil, fmt.Errorf("deployment server %q is not configured", chall.ServerDeployed)
		}
		containers, err = remoteManager.SearchContainerByFilterRemote(map[string]string{"id": chall.ContainerId}, server)
	}
	if err != nil {
		return nil, fmt.Errorf("search for container %s: %w", chall.ContainerId, err)
	}
	if len(containers) > 1 {
		return nil, errors.New("multiple containers matched the challenge container ID")
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("Underlying container for getting log is not present.")
	}

	if live {
		if config.Cfg.UseLocalDockerDaemon(chall.ServerDeployed) {
			if err := cr.ShowLiveContainerLogs(chall.ContainerId); err != nil {
				return nil, err
			}
		} else {
			server := config.Cfg.AvailableServers[chall.ServerDeployed]
			if err := remoteManager.ShowLiveContainerLogsRemote(chall.ContainerId, server); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}

	if config.Cfg.UseLocalDockerDaemon(chall.ServerDeployed) {
		return cr.GetContainerStdLogs(chall.ContainerId)
	}

	server := config.Cfg.AvailableServers[chall.ServerDeployed]
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
