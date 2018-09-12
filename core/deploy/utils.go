package deploy

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/fristonio/beast/core"
	log "github.com/sirupsen/logrus"
)

func ValidateChallengeConfig(challengeDir string) error {
	configFile := filepath.Join(challangeDir, core.CONFIG_FILE_NAME)

	if _, err := os.Stat(configFile); err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Config file for %s does not exist", challengeDir)
			return errors.New("beast.toml file does not exist")
		} else {
			log.Warnf("Requested challenge config(%s) is not accessbile", configFile)
			return errors.New("Not accessible directory config.")
		}
	}

	var config core.BeastConfig
	_, err := toml.DecodeFile(configFile, config)
	if err != nil {
		return err
	}

	return nil
}

func ValidateChallengeDir(challengeDir string) error {
	log.Debugf("Validating Directory : %s", challengeDir)

	// Check if the provided path exist
	if dirPath, err := os.Stat(challengeDir); err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Requested challenge Directory(%s) does not exist", challengeDir)
			return errors.New("Directory does not exist")
		} else {
			log.Warnf("Requested challenge Directory(%s) is not accessbile", challengeDir)
			return errors.New("Not accessible directory.")
		}
	}

	// Check if the path provided points to a directory
	if !dirPath.IsDir() {
		log.Warnf("%s is not a directory", challengeDir)
		return errors.New("Not a directory")
	}

	err = ValidateChallengeConfig(challengeDir)
	if err != nil {
		return err
	}

	return nil
}
