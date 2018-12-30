package manager

import (
	"fmt"
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/sidecar"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

func configureSidecar(config *cfg.BeastChallengeConfig) error {
	log.Infof("Configuring sidecar for challenge : %s", config.Challenge.Metadata.Name)

	sidecarAgent, err := sidecar.GetSidecarAgent(config.Challenge.Metadata.Sidecar)
	if err != nil {
		return err
	}

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, config.Challenge.Metadata.Name)
	configPath := filepath.Join(stagingDir, fmt.Sprintf(".%s.env", config.Challenge.Metadata.Sidecar))
	err = utils.ValidateFileExists(configPath)
	if err == nil {
		log.Infof("Sidecar configuration file already exists, not creating a new.")
		return nil
	}

	err = sidecarAgent.Bootstrap(configPath)
	if err != nil {
		return fmt.Errorf("Error while bootstrapping sidecar configuration: %s", err)
	}

	log.Infof("Sidecar configuration bootstrap complete.")
	return nil
}

func cleanSidecar(config *cfg.BeastChallengeConfig) error {
	log.Infof("Destroying the sidecar configuration for challenge: %s", config.Challenge.Metadata.Name)

	sidecarAgent, err := sidecar.GetSidecarAgent(config.Challenge.Metadata.Sidecar)
	if err != nil {
		return err
	}

	stagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, config.Challenge.Metadata.Name)
	configPath := filepath.Join(stagingDir, fmt.Sprintf(".%s.env", config.Challenge.Metadata.Sidecar))
	err = utils.ValidateFileExists(configPath)
	if err != nil {
		log.Warnf("Sidecar configuration does not exist, nothing to wipe.")
		return nil
	}

	err = sidecarAgent.Destroy(configPath)
	if err != nil {
		return fmt.Errorf("Error while destroying sidecar configuration: %s", err)
	}

	log.Infof("Sidecar configuration cleanup complete.")
	return nil
}
