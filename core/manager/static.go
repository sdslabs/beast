package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
// The port for the deployment of the static container is specified in the variable
// BEAST_CHALLENGES_STATIC_PORT, this port should be free and will be the port on which
// nginx container for static files will be running.
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
	if len(images) == 0 {
		log.Debugf("Static content image does not exist, build image manually")
		return errors.New("IMAGE_NOT_FOUND_ERROR")
	}

	// Remove the prefix sha256:
	imageId := images[0].ID[7:]
	stagingDirPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	err = utils.CreateIfNotExistDir(stagingDirPath)
	if err != nil {
		log.Errorf("Error in validating staging mount point : %s", err)
		return errors.New("INVALID_STAGING_AREA")
	}

	beastStaticAuthFile := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STATIC_AUTH_FILE)
	err = utils.ValidateFileExists(beastStaticAuthFile)
	if err != nil {
		p := fmt.Errorf("BEAST STATIC: Authentication file does not exist for beast static container, cannot proceed deployment")
		log.Error(p.Error())
		return p
	}

	staticMount := make(map[string]string)
	staticMount[stagingDirPath] = core.BEAST_STAGING_AREA_MOUNT_POINT
	staticMount[beastStaticAuthFile] = filepath.Join("/", core.BEAST_STATIC_AUTH_FILE)
	portMap := cr.PortMapping{
		ContainerPort: core.BEAST_CHALLENGES_STATIC_PORT,
		HostPort:      core.BEAST_CHALLENGES_STATIC_PORT,
	}

	containerConfig := cr.CreateContainerConfig{
		PortMapping:   []cr.PortMapping{portMap},
		MountsMap:     staticMount,
		ImageId:       imageId,
		ContainerName: core.BEAST_STATIC_CONTAINER_NAME,
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
func UndeployStaticContentContainer() {
	err := coreutils.CleanupContainerByFilter("name", core.BEAST_STATIC_CONTAINER_NAME)
	if err != nil {
		log.Errorf("Error while cleaning old static content container : %s", err)
	} else {
		log.Infof("Static content container undeployed")
	}
}

// Deploy a static challenge
func DeployStaticChallenge(challConf *cfg.BeastChallengeConfig, challenge *database.Challenge, challengeDir string) {
	log.Infof("Starting static challenge deploy pipeline")
	challengeStagingRoot := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challConf.Challenge.Metadata.Name)
	challengeStagingDir := filepath.Join(challengeStagingRoot, core.BEAST_STATIC_FOLDER)

	// Check if the challenge is already in staged state
	// Remove the already staged challenge and then copy the new files.
	err := utils.ValidateDirExists(challengeStagingDir)
	if err == nil {
		err = utils.RemoveDirRecursively(challengeStagingDir)
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

		database.UpdateChallenge(challenge, map[string]interface{}{"Status": core.DEPLOY_STATUS["deployed"]})

		configFile := filepath.Join(challengeStagingDir, core.CHALLENGE_CONFIG_FILE_NAME)
		err = os.Rename(configFile, filepath.Join(challengeStagingRoot, core.CHALLENGE_CONFIG_FILE_NAME))
		if err != nil {
			log.Errorf("Error while removing challenge config file.")
		}
	}
}
