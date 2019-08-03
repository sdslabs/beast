package mysql

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	coreutils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

type MySQLDeployer struct{}

const MYSQL_SIDECAR_PORT uint32 = 9500

func (a *MySQLDeployer) DeploySidecar(config *cr.CreateContainerConfig) error {
	err := coreutils.CleanupContainerByFilter("name", core.MYSQL_SIDECAR_HOST)
	if err != nil {
		log.Errorf("Error while cleaning old MySQL container : %s", err)
		return errors.New("CLEANUP_ERROR")
	}

	images, err := cr.SearchImageByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", core.MYSQL_SIDECAR_HOST)})
	if len(images) == 0 {
		log.Debugf("MySQL image does not exist, build image manually")
		return errors.New("IMAGE_NOT_FOUND_ERROR")
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

	staticMount := make(map[string]string)
	staticMount[stagingDirPath] = core.BEAST_STAGING_AREA_MOUNT_POINT
	staticMount[beastStaticAuthFile] = filepath.Join("/", core.BEAST_STATIC_AUTH_FILE)
	port := []uint32{MYSQL_SIDECAR_PORT}
	mysqlRootPassword := coreUtils.RandString(8)
	m := map[string]string{
		"MYSQL_ROOT_PASSWORD": mysqlRootPassword,
	}

	count := len(m)
	all := make([]string, count*2)

	containerConfig := cr.CreateContainerConfig{
		PortsList:        port,
		ImageId:          imageId,
		ContainerName:    "mysql",
		ContainerNetwork: "beast-mysql",
		ContainerEnv:     all,
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

	log.Infof("MySQL CONTAINER deployed and started : %s", containerId)

	return nil
	return nil
}

func (a *MySQLDeployer) UndeploySidecar(config *cr.CreateContainerConfig) error {
	err := coreutils.CleanupContainerByFilter("name", core.MYSQL_SIDECAR_HOST)
	if err != nil {
		log.Errorf("Error while cleaning old MySQL container : %s", err)
	} else {
		log.Infof("MySQL container undeployed")
	}
	return nil
}
