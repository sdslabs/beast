package utils

import (
	"fmt"

	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"

	log "github.com/sirupsen/logrus"
)

func CleanupContainerByFilter(filter, filterVal string) error {
	if filter != "id" && filter != "name" {
		return fmt.Errorf("Not a valid filter %s", filter)
	}

	containers, err := cr.SearchContainerByFilter(map[string]string{filter: filterVal})
	if err != nil {
		log.Error("Error while searching for container with %s : ", filter, filterVal)
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

	if len(erroredContainers) != 0 {
		return fmt.Errorf("Error while cleaning up container : %s", erroredContainers)
	}

	return nil
}

func CleanupChallengeContainers(chall *database.Challenge, config cfg.BeastChallengeConfig) error {
	if IsContainerIdValid(chall.ContainerId) {
		err := CleanupContainerByFilter("id", chall.ContainerId)
		if err != nil {
			return err
		}

		database.UpdateChallenge(chall, map[string]interface{}{"ContainerId": GetTempContainerId(chall.Name)})
	}

	err := CleanupContainerByFilter("name", EncodeID(config.Challenge.Metadata.Name))
	return err
}

func CleanupChallengeImage(chall *database.Challenge) error {
	err := cr.RemoveImage(chall.ImageId)
	if err != nil {
		log.Error("Error while cleaning up image with id ", chall.ImageId)
		return err
	}

	database.UpdateChallenge(chall, map[string]interface{}{"ImageId": GetTempImageId(chall.Name)})

	return nil
}

func CleanupChallengeIfExist(config cfg.BeastChallengeConfig) error {
	chall, err := database.QueryFirstChallengeEntry("name", config.Challenge.Metadata.Name)
	if err != nil {
		log.Errorf("Error while database query for challenge %s", config.Challenge.Metadata.Name)
		return err
	}

	if chall.Name == "" {
		log.Info("No such challenge exist in the database")
		return nil
	}

	err = CleanupChallengeContainers(&chall, config)
	if err != nil {
		return fmt.Errorf("Error while cleaning up the container : %v", err)
	}

	if !IsImageIdValid(chall.Name) {
		log.Warn("Looks like we don't have the image ID in database for challenge, Nothing to remove")
		return nil
	}
	err = CleanupChallengeImage(&chall)
	return err
}
