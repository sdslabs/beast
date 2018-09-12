package deploy

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/fristonio/beast/core"
	"github.com/fristonio/beast/utils"
	log "github.com/sirupsen/logrus"
)

func ValidateChallengeConfig(challengeDir string) error {
	configFile := filepath.Join(challengeDir, core.CONFIG_FILE_NAME)

	log.Debug("Checking beast.toml file existance validity")
	err := utils.ValidateFileExists(configFile)
	if err != nil {
		return err
	}

	var config core.BeastConfig
	_, err = toml.DecodeFile(configFile, &config)
	if err != nil {
		return err
	}

	err = config.ValidateRequiredFields()
	if err != nil {
		return err
	}

	log.Debug("Challenge config file beast.toml is valid")
	return nil
}

// Validates a directory which is considered as a challenge directory
// The function returns an error if the directory is not valid or if it
// does not have valid structure required by beast.
func ValidateChallengeDir(challengeDir string) error {
	log.Debugf("Validating Directory : %s", challengeDir)

	err := utils.ValidateDirExists(challengeDir)
	if err != nil {
		return err
	}

	err = ValidateChallengeConfig(challengeDir)
	if err != nil {
		return err
	}

	log.Infof("Challenge directory validated")
	return nil
}
