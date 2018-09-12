package core

import (
	"errors"

	log "github.com/sirupsen/logrus"
)

const (
	CONFIG_FILE_NAME       string = "beast.toml"
	DEFAULT_CHALLENGE_NAME string = "Backdoor-Challenge"
	DEFAULT_AUTHOR_NAME    string = "ghost"
)

// This is the beast challenge config file structure
// any other field specified in the file other than this structure
// will be ignored.
//
// Take a look at template beast.toml file in templates package
// to see how to specify the file and what all fields are available.
type BeastConfig struct {
	Challenge Challenge `toml:"challenge"`
	Author    Author    `toml:"author"`
}

func (config *BeastConfig) ValidateRequiredFields() error {
	log.Debugf("Validating BeastConfig required fields")
	err := config.Challenge.ValidateRequiredFields()
	if err != nil {
		log.Debugf("Error while validating `Challenge` required fields : %s", err.Error())
		return err
	}

	err = config.Author.ValidateRequiredFields()
	if err != nil {
		log.Debugf("Error while validating `Author`'s required fields : %s", err.Error())
		return err
	}

	log.Debugf("BeastConfig required fields validated")
	return nil
}

type Challenge struct {
	Id               string           `toml:"id"`
	Name             string           `toml:"name"`
	ChallengeType    string           `toml:"challenge_type"`
	ChallengeDetails ChallengeDetails `toml:"details"`
}

func (config *Challenge) ValidateRequiredFields() error {
	if config.Id == "" || config.ChallengeType == "" {
		return errors.New("Challenge `id` and `challenge_type` are required")
	}

	if config.Name == "" {
		config.Name = DEFAULT_CHALLENGE_NAME
	}

	err := config.ChallengeDetails.ValidateRequiredFields()
	if err != nil {
		log.Debugf("Error while validating `ChallengeDetails`'s required fields : %s", err.Error())
		return err
	}

	return nil
}

type ChallengeDetails struct {
	Flag             string   `toml:"flag"`
	AptDeps          []string `toml:"apt_dependencies"`
	SetupScript      string   `toml:"setup_script"`
	StaticContentDir string   `toml:"static_content_dir"`
	RunCmd           string   `toml:"run_cmd"`
}

func (config *ChallengeDetails) ValidateRequiredFields() error {
	if config.Flag == "" || config.RunCmd == "" {
		return errors.New("Challenge `flag` and `run_cmd` are required")
	}

	return nil
}

type Author struct {
	Name   string `toml:"name"`
	Email  string `toml:"email"`
	SSHKey string `toml:"ssh_key"`
}

func (config *Author) ValidateRequiredFields() error {
	if config.Email == "" || config.SSHKey == "" {
		return errors.New("Challenge `email` and `ssh_key` are required")
	}

	if config.Name == "" {
		config.Name = DEFAULT_AUTHOR_NAME
	}

	return nil
}
