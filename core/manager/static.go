package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreutils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

// Deploy the static content container for beast
// The image for the static container should be prebuilt, which can be found
// in /extras/static-content/ of the root of the project
// The image name for the static content docker image shoule be specified in the
// BEAST_STATIC_CONTAINER_NAME:latest variable
// This function does not build the image for static containers.
// The static container receives a separate read-only mount for each challenge's public
// assets. Other staging data is never visible to the web server.
//
// Each challenges have its own static file folder inside the challenges directory.
// The whole staging area of beast configuration is mounted on the docker container
// to serve the static files to the user. The location of the static content for each
// challenge for staging area is $BEAST_ROOT/staging/$CHALLENGE/static
// This directory is automatically populated with the desired challenge static files
// when the challenge is commanded to be staged.
func DeployStaticContentContainer() error {
	err := coreutils.CleanupContainerByFilter("name", core.BEAST_STATIC_CONTAINER_NAME)
	if err != nil {
		log.Errorf("Error while cleaning old static content container : %s", err)
		return errors.New("CLEANUP_ERROR")
	}

	images, err := cr.SearchImageByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", core.BEAST_STATIC_CONTAINER_NAME)})
	if err != nil {
		log.Errorf("Error searching for static content image: %s", err)
		return errors.New("IMAGE_SEARCH_ERROR")
	}
	if len(images) == 0 {
		log.Debugf("Static content image does not exist, build image manually")
		return errors.New("IMAGE_NOT_FOUND_ERROR")
	}

	imageId := strings.TrimPrefix(images[0].ID, "sha256:")

	staticMount := make(map[string]string)
	challenges, err := database.QueryAllChallenges()
	if err != nil {
		return fmt.Errorf("query static challenge assets: %w", err)
	}
	for _, challenge := range challenges {
		source := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challenge.Name, core.BEAST_STATIC_FOLDER)
		info, statErr := os.Lstat(source)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("invalid static asset directory for %s", challenge.Name)
		}
		staticMount[source] = filepath.Join(core.BEAST_STAGING_AREA_MOUNT_POINT, challenge.Name, core.BEAST_STATIC_FOLDER)
	}
	portMap := cr.PortMapping{
		ContainerPort: core.BEAST_STATIC_CONTAINER_PORT,
		HostPort:      core.BEAST_CHALLENGES_STATIC_PORT,
	}

	containerConfig := cr.CreateContainerConfig{
		PortMapping:   []cr.PortMapping{portMap},
		MountsMap:     staticMount,
		ImageId:       imageId,
		ContainerName: core.BEAST_STATIC_CONTAINER_NAME,
		CPUShares:     cfg.Cfg.CPUShares,
		CPUsLimit:     cfg.Cfg.CPUsLimit,
		Memory:        cfg.Cfg.Memory,
		PidsLimit:     cfg.Cfg.PidsLimit,
	}
	containerId, err := cr.CreateContainerFromImage(&containerConfig)
	if err != nil {
		if containerId != "" {
			log.Errorf("Error while starting the container : %s", err)
			return errors.New("CONTAINER_ERROR")
		}

		log.Errorf("Error while trying to create a container for the challenge: %s", err)
		return errors.New("CONTAINER_ERROR")
	}

	log.Infof("STATIC CONTAINER deployed and started : %s", containerId)

	return nil
}

// This cleans up the container deployed by DeployStaticContentContainer function
// The image is preserved after calling the function and thus need not be build again.
func UndeployStaticContentContainer() error {
	err := coreutils.CleanupContainerByFilter("name", core.BEAST_STATIC_CONTAINER_NAME)
	if err != nil {
		log.Errorf("Error while cleaning old static content container : %s", err)
		return err
	}
	log.Infof("Static content container undeployed")
	return nil
}

// Deploy a static challenge
func DeployStaticChallenge(challConf *cfg.BeastChallengeConfig, challenge *database.Challenge, challengeDir string) {
	log.Infof("Starting static challenge deploy pipeline")
	challengeStagingRoot := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challConf.Challenge.Metadata.Name)
	// challengeStagingDir := filepath.Join(challengeStagingRoot, core.BEAST_STATIC_FOLDER)

	// Check if the challenge is already in staged state
	// Remove the already staged challenge and then copy the new files.
	err := utils.ValidateDirExists(challengeStagingRoot)
	if err == nil {
		err = utils.RemoveDirRecursively(challengeStagingRoot)
		if err != nil {
			log.Errorf("Error while cleaning already staged static challenge %s : %s", challConf.Challenge.Metadata.Name, err)
			return
		}
	}

	err = utils.CopyDirectory(challengeDir, challengeStagingRoot)
	if err != nil {
		log.Errorf("Error while copying to the staging directory: %s", err)
	} else {
		log.Infof("Challenge %s has been deployed as a static challenge", challConf.Challenge.Metadata.Name)

		database.UpdateChallenge(challenge, map[string]interface{}{"status": core.DEPLOY_STATUS["deployed"]})

		// configFile := filepath.Join(challengeStaticDir, core.CHALLENGE_CONFIG_FILE_NAME)
		// err = os.Rename(configFile, filepath.Join(challengeStagingRoot, core.CHALLENGE_CONFIG_FILE_NAME))
		// if err != nil {
		// 	log.Errorf("Error while removing challenge config file: %s", err)
		// }

	}
}
