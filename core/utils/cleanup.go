package utils

import (
	"fmt"
	container_types "github.com/docker/docker/api/types"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

func CleanupContainerByFilter(filter, filterVal string) error {
	if filter != "id" && filter != "name" && filter != "label" {
		return fmt.Errorf("Not a valid filter %s", filter)
	}
	var containers []container_types.Container
	var err error
	containers, err = cr.SearchContainerByFilter(map[string]string{filter: filterVal})
	if err != nil {
		log.Errorf("Error while searching for container with %s : %s", filter, filterVal)
		return err
	}
	server := config.AvailableServer{}
	remoteContainers, err := remoteManager.SearchContainerByFilterRemote(map[string]string{filter: filterVal}, server)
	if err != nil {
		log.Errorf("Error while searching for remote container with %s : %s", filter, filterVal)
		return err
	}

	var erroredContainers []string
	if len(containers) != 0 {
		log.Infof("Cleaning up container with %s %s", filter, filterVal)
		for _, container := range containers {
			err = cr.StopAndRemoveContainer(container.ID)
			if err != nil {
				erroredContainers = append(erroredContainers, container.ID)
				log.Errorf("Error while cleaning up container %s : %s", container.ID, err)
			}
		}
	}

	if len(remoteContainers) != 0 {
		log.Infof("Cleaning up remote container with %s %s", filter, filterVal)
		for _, container := range remoteContainers {
			err = remoteManager.StopAndRemoveContainerRemote(container.ID, config.AvailableServer{})
			if err != nil {
				erroredContainers = append(erroredContainers, container.ID)
				log.Errorf("Error while cleaning up remote container %s : %s", container.ID, err)
			}
		}
	}

	if len(erroredContainers) != 0 {
		return fmt.Errorf("Error while cleaning up container : %s", erroredContainers)
	}
	return nil
}

func CleanupChallengeContainers(chall *database.Challenge, config cfg.BeastChallengeConfig) error {
	if chall.DeploymentType == core.DEPLOYMENT_TYPES["docker_compose"] {
		log.Debugf("Cleaning up Docker Compose challenge: %s", chall.Name)
		// Same -p as deployPipeline / ComposeDown for non-instanced compose.
		projectName := utils.ProjectNameNotInstanced(chall.Name)

		if !cfg.Cfg.UseLocalDockerDaemon(chall.ServerDeployed) {
			server := cfg.Cfg.AvailableServers[chall.ServerDeployed]
			if err := remoteManager.ComposeDownProjectRemote(projectName, server); err != nil {
				log.Errorf("Error running docker compose down on remote: %v", err)
				return err
			}
		} else if err := cr.ComposeDownProject(projectName); err != nil {
			log.Errorf("Error running docker compose down locally: %v", err)
			return err
		}

		database.UpdateChallenge(chall, map[string]any{"ContainerId": GetTempContainerId(chall.Name)})
		if err := cache.FreeContainerPortsOnHost(chall.ServerDeployed, projectName); err != nil {
			log.Warnf("Failed to free ports for compose challenge %s: %v", chall.Name, err)
		}
		return nil
	}

	if IsContainerIdValid(chall.ContainerId) {
		err := CleanupContainerByFilter("id", chall.ContainerId)
		if err != nil {
			return err
		}

		database.UpdateChallenge(chall, map[string]any{"ContainerId": GetTempContainerId(chall.Name)})
		if err := cache.FreeContainerPortsOnHost(chall.ServerDeployed, chall.ContainerId); err != nil {
			log.Warnf("Failed to free ports for challenge %s: %v", chall.Name, err)
		}
	}

	err := CleanupContainerByFilter("name", utils.EncodeID(config.Challenge.Metadata.Name))
	return err
}

func CleanupChallengeImage(chall *database.Challenge) error {
	if cfg.Cfg.UseLocalDockerDaemon(chall.ServerDeployed) {
		err := cr.RemoveImage(chall.ImageId)
		if err != nil {
			log.Errorf("Error while cleaning up image with id %s", chall.ImageId)
			return err
		}
	} else {
		server := config.Cfg.AvailableServers[chall.ServerDeployed]
		err := remoteManager.RemoveImageRemote(chall.ImageId, server)
		if err != nil {
			log.Errorf("Error while cleaning up image on remote %s with id %s", chall.ServerDeployed, chall.ImageId)
			return err
		}
	}

	database.UpdateChallenge(chall, map[string]any{"ImageId": GetTempImageId(chall.Name)})

	return nil
}

func CleanupChallengeIfExist(config cfg.BeastChallengeConfig) error {
	chall, found, err := database.FindFirstChallengeEntry("name", config.Challenge.Metadata.Name)
	if err != nil {
		log.Errorf("Error while database query for challenge %s", config.Challenge.Metadata.Name)
		return err
	}

	if !found {
		log.Info("No such challenge exist in the database")
		return nil
	}

	err = CleanupChallengeContainers(&chall, config)
	if err != nil {
		return fmt.Errorf("Error while cleaning up the container : %v", err)
	}

	if !IsImageIdValid(chall.ImageId) {
		log.Warn("Looks like we don't have the image ID in database for challenge, Nothing to remove")
		return nil
	}
	err = CleanupChallengeImage(&chall)
	return err
}
