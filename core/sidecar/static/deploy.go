package static

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	coreutils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

type StaticDeployer struct{}

const STATIC_SIDECAR_PORT uint32 = 8080

func (a *StaticDeployer) DeploySidecar() error {
	images, err := cr.SearchImageByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", core.STATIC_SIDECAR_HOST)})
	if len(images) == 0 {
		log.Debugf("Static image does not exist, building image")
		imageLocation := filepath.Join(core.BEAST_REMOTES_DIR, ".beast/extras/static-content/")
		buff, imageID, err := cr.BuildImageFromTarContext(core.STATIC_SIDECAR_HOST, "", imageLocation)
		if buff == nil || err != nil {
			return errors.New("IMAGE_NOT_FOUND_ERROR")
		}
		log.Infof("Image ID of image : %s", imageID)
	}

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

	container, err := cr.SearchContainerByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", core.STATIC_SIDECAR_HOST)})
	if len(container) != 0 {
		log.Infof("Container for static sidecar with name beast-static already exists.")
		return nil
	}

	staticMount := make(map[string]string)
	staticMount[stagingDirPath] = core.BEAST_STAGING_AREA_MOUNT_POINT
	staticMount[beastStaticAuthFile] = filepath.Join("/", core.BEAST_STATIC_AUTH_FILE)
	port := []uint32{STATIC_SIDECAR_PORT}

	containerConfig := cr.CreateContainerConfig{
		PortsList:        port,
		ImageId:          imageId,
		MountsMap:        staticMount,
		ContainerName:    "beast-static",
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

	log.Infof("Beast-static CONTAINER deployed and started : %s", containerId)

	return nil
}

func (a *StaticDeployer) UndeploySidecar() error {
	err := coreutils.CleanupContainerByFilter("name", core.STATIC_SIDECAR_HOST)
	if err != nil {
		log.Errorf("Error while cleaning old beast-static container : %s", err)
	} else {
		log.Infof("Beast-static container undeployed")
	}
	return nil
}
