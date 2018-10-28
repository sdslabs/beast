package config

import (
	"errors"
	"fmt"

	"github.com/sdslabs/beastv4/core"

	log "github.com/sirupsen/logrus"
)

// This is the beast challenge config file structure
// any other field specified in the file other than this structure
// will be ignored.
//
// Take a look at template beast.toml file in templates package
// to see how to specify the file and what all fields are available.
type BeastChallengeConfig struct {
	Challenge Challenge `toml:"challenge"`
	Author    Author    `toml:"author"`
}

func (config *BeastChallengeConfig) ValidateRequiredFields() error {
	log.Debugf("Validating BeastChallengeConfig required fields")
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

	log.Debugf("BeastChallengeConfig required fields validated")
	return nil
}

// This structure contains information related to challenge,
// Challenge Metadata
//
// * Name - Name of the challenge(This must be unique and will correspond to identifier)
// * ChallengeType - Type of the challenge(web/service/ssh)
// * ChallengeDetails - Another substructure cotaining details about challenge
type Challenge struct {
	Name             string           `toml:"name"`
	ChallengeType    string           `toml:"challenge_type"`
	ChallengeDetails ChallengeDetails `toml:"details"`
}

func (config *Challenge) ValidateRequiredFields() error {
	if config.ChallengeType == "" || config.Name == "" {
		return errors.New("Challenge `id`, `name` and `challenge_type` are required")
	}

	err := config.ChallengeDetails.ValidateRequiredFields()
	if err != nil {
		log.Debugf("Error while validating `ChallengeDetails`'s required fields : %s", err.Error())
		return err
	}

	return nil
}

// This contains challenge specific properties which includes
//
// * Flag - Flag corresponding to the challenge
// * AptDeps - Apt dependencies for the challenge
// * SetupScript - relative path to the challenge setup script
// * StaticContentDir - Relative path to the directory which you want
// 		to serve statically for the challenge, for example a libc for binary
// 		challenge.
// * RunCmd - Command to run to start the challenge.
type ChallengeDetails struct {
	Flag                    string   `toml:"flag"`
	AptDeps                 []string `toml:"apt_dependencies"`
	SetupScript             []string `toml:"setup_script"`
	StaticContentDir        string   `toml:"static_content_dir"`
	StaticContentServerPort uint32   `toml:"static_content_port"`
	RunCmd                  string   `toml:"run_cmd"`
	Ports                   []uint32 `toml:"ports"`
}

func (config *ChallengeDetails) ValidateRequiredFields() error {
	if config.Flag == "" {
		return errors.New("Challenge `flag` and `run_cmd` are required")
	}

	if len(config.Ports) > int(core.MAX_PORT_PER_CHALL) {
		return fmt.Errorf("Max ports allowed for challenge : %d given : %d", core.MAX_PORT_PER_CHALL, len(config.Ports))
	}

	if config.StaticContentDir != "" {
		if config.StaticContentServerPort == 0 || len(config.Ports) == 0 {
			return errors.New("Valid static content server port should be provided")
		}

		validPortMap := false
		for _, port := range config.Ports {
			if port == config.StaticContentServerPort {
				validPortMap = true
			}
		}

		if !validPortMap {
			return errors.New("Static content dir server port should also be present in Ports array")
		}
	}

	return nil
}

// Metadata related to author of the challenge, this structure includes
//
// * Name - Name of the author of the challenge
// * Email - Email of the author
// * SSHKey - Public SSH key for the challenge author, to give the access
//		to the challenge container.
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
		config.Name = core.DEFAULT_AUTHOR_NAME
	}

	return nil
}
