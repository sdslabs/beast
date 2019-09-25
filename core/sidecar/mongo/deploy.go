package mongo

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

type MongoDeployer struct{}

const MONGO_SIDECAR_PORT uint32 = 9501

func (a *MongoDeployer) DeploySidecar() error {
	images, err := cr.SearchImageByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", core.MONGO_SIDECAR_HOST)})
	if len(images) == 0 {
		log.Debugf("Mongo image does not exist, build image manually")
		imageLocation := filepath.Join(core.BEAST_REMOTES_DIR, ".beast/remote/temp/challenges/mongo/")
		buff, imageID, err := cr.BuildImageFromTarContext("mongo", "", imageLocation)
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

	networkList, err := cr.SearchNetworkByFilter(map[string]string{"networkName": "beast-mongo"})
	if networkList == nil || err != nil {
		log.Warnf("No Mongo network found. Creating one")
		networkconfig := &cr.CreateNetworkConfig{
			NetworkName: "beast-mongo",
		}
		network, err := cr.CreateNetwork(networkconfig)
		if network == "" || err != nil {
			log.Errorf("Error in creating beast network.")
			return nil
		}
	}

	container, err := cr.SearchContainerByFilter(map[string]string{"reference": fmt.Sprintf("%s:latest", core.MONGO_SIDECAR_HOST)})
	if len(container) != 0 {
		log.Infof("Container for mongo sidecar with name Mongo already exists.")
		return nil
	}

	staticMount := make(map[string]string)
	staticMount[stagingDirPath] = core.BEAST_STAGING_AREA_MOUNT_POINT
	staticMount[beastStaticAuthFile] = filepath.Join("/", core.BEAST_STATIC_AUTH_FILE)
	port := []uint32{MONGO_SIDECAR_PORT}
	mongoRootPassword := coreUtils.RandString(8)
	mongoRootUsername := coreUtils.RandString(8)
	m := map[string]string{
		"MONGO_INITDB_ROOT_USERNAME": mongoRootPassword,
		"MONGO_INITDB_ROOT_PASSWORD": mongoRootUsername,
	}

	count := len(m)
	all := make([]string, count*2)

	containerConfig := cr.CreateContainerConfig{
		PortsList:        port,
		ImageId:          imageId,
		ContainerName:    "mongo",
		ContainerNetwork: "beast-mongo",
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

	log.Infof("Mongo CONTAINER deployed and started : %s", containerId)

	return nil
	return nil
}

func (a *MongoDeployer) UndeploySidecar() error {
	err := coreutils.CleanupContainerByFilter("name", core.MONGO_SIDECAR_HOST)
	if err != nil {
		log.Errorf("Error while cleaning old Mongo container : %s", err)
	} else {
		log.Infof("Mongo container undeployed")
	}
	return nil
}
