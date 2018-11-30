package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

var AVAILABLE_CHALLENGE_TYPES = []string{"web", "service", core.STATIC_CHALLENGE_TYPE_NAME}

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
// * ChallengeEnv - Challenge environment configuration variables
// * ChallengeMetadata - Challenge Metadata configuration variables
type Challenge struct {
	Metadata ChallengeMetadata `toml:"metadata"`
	Env      ChallengeEnv      `toml:"env"`
}

func (config *Challenge) ValidateRequiredFields() error {
	err, staticChall := config.Metadata.ValidateRequiredFields()
	if err != nil {
		log.Debugf("Error while validating `ChallengeMetadata`'s required fields : %s", err.Error())
		return err
	} else if staticChall {
		log.Debugf("Challenge provided is a static challenge.")
		return nil
	}

	err = config.Env.ValidateRequiredFields()
	if err != nil {
		log.Debugf("Error while validating `ChallengeEnv`'s required fields : %s", err.Error())
		return err
	}

	return nil
}

// This contains challenge specific properties which includes
//
// * AptDeps - Apt dependencies for the challenge
// * SetupScripts - relative path to the challenge setup scripts
// * StaticContentDir - Relative path to the directory which you want
// 		to serve statically for the challenge, for example a libc for binary
// 		challenge.
// * RunCmd - Command to run or start the challenge.
// * Base for the challenge, this might be extension to dockerfile usage
// 		like for a php challenge this can be php:web, for node node:web
// 		for xinetd services xinetd:service
// * Ports: A list of ports to be used by the challenge.
type ChallengeMetadata struct {
	Flag string `toml:"flag"`
	Name string `toml:"name"`
	Type string `toml:"type"`
}

func (config *ChallengeMetadata) ValidateRequiredFields() (error, bool) {
	if config.Name == "" || config.Flag == "" {
		return fmt.Errorf("Name and Flag required for the challenge"), false
	}

	// Check if the config type is static here and if it is
	// then return an indication for that, so that caller knows if it need
	// to check a valid environment or not.
	for i := range AVAILABLE_CHALLENGE_TYPES {
		if AVAILABLE_CHALLENGE_TYPES[i] == config.Type {
			if config.Type == core.STATIC_CHALLENGE_TYPE_NAME {
				// Challenge is a standalone static challenge
				return nil, true
			}

			return nil, false
		}
	}

	return fmt.Errorf("Not a valid challenge type : %s", config.Type), false
}

type ChallengeEnv struct {
	AptDeps          []string `toml:"apt_deps"`
	Ports            []uint32 `toml:"ports"`
	SetupScripts     []string `toml:"setup_scripts"`
	StaticContentDir string   `toml:"static_dir"`
	RunCmd           string   `toml:"run_cmd"`
	Base             string   `toml:"base"`
	BaseImage        string   `toml:"base_image"`
}

func (config *ChallengeEnv) ValidateRequiredFields() error {
	if len(config.Ports) == 0 {
		return errors.New("Are you sure you have specified the ports used by challenge")
	}

	if len(config.Ports) > int(core.MAX_PORT_PER_CHALL) {
		return fmt.Errorf("Max ports allowed for challenge : %d given : %d", core.MAX_PORT_PER_CHALL, len(config.Ports))
	}

	if config.StaticContentDir != "" {
		if filepath.IsAbs(config.StaticContentDir) {
			return fmt.Errorf("Static content directory path should be relative to challenge directory root")
		}
	}

	if config.RunCmd == "" {
		return fmt.Errorf("A valid run_cmd should be provided for the challenge environment")
	}

	if config.BaseImage == "" {
		config.BaseImage = core.DEFAULT_BASE_IMAGE
	}

	if !utils.StringInSlice(config.BaseImage, Cfg.AllowedBaseImages) {
		return fmt.Errorf("The base image: %s is not supported", config.BaseImage)
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
