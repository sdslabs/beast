package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

const SERVICE_CONTAINER_DEPS string = "xinetd"
const SERVICE_CHALL_RUN_CMD string = "xinetd -dontfork"

// This is the beast challenge config file structure
// any other field specified in the file other than this structure
// will be ignored.
//
// Take a look at template beast.toml file in templates package
// to see how to specify the file and what all fields are available.
type BeastChallengeConfig struct {
	Challenge Challenge `toml:"challenge"`
	Author    Author    `toml:"author"`
	Resources Resources `toml:"resource"`
}

func (config *BeastChallengeConfig) ValidateRequiredFields(challdir string) error {
	log.Debugf("Validating BeastChallengeConfig required fields")
	err := config.Challenge.ValidateRequiredFields(challdir)
	if err != nil {
		log.Debugf("Error while validating `Challenge` required fields : %s", err.Error())
		return err
	}

	err = config.Author.ValidateRequiredFields()
	if err != nil {
		log.Debugf("Error while validating `Author`'s required fields : %s", err.Error())
		return err
	}

	config.Resources.ValidateRequiredFields()

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

func (config *Challenge) ValidateRequiredFields(challdir string) error {
	err, staticChall := config.Metadata.ValidateRequiredFields()
	if err != nil {
		log.Debugf("Error while validating `ChallengeMetadata`'s required fields : %s", err.Error())
		return err
	} else if staticChall {
		log.Debugf("Challenge provided is a static challenge.")
		return nil
	}

	err = config.Env.ValidateRequiredFields(config.Metadata.Type, challdir)
	if err != nil {
		log.Debugf("Error while validating `ChallengeEnv`'s required fields : %s", err.Error())
		return err
	}

	return nil
}

// This contains challenge meta data
//
// * Flag - Apt dependencies for the challenge
// * Name - relative path to the challenge setup scripts
// * Type - Relative path to the directory which you want
type ChallengeMetadata struct {
	Flag        string   `toml:"flag"`
	Name        string   `toml:"name"`
	Type        string   `toml:"type"`
	Tags        []string `toml:"tags"`
	Sidecar     string   `toml:"sidecar"`
	Description string   `toml:"description"`
	Hint        string   `toml:"hint"`
}

// In this validation returned boolean value represents if the challenge type is
// static or not.
func (config *ChallengeMetadata) ValidateRequiredFields() (error, bool) {
	if config.Name == "" || config.Flag == "" {
		return fmt.Errorf("Name and Flag required for the challenge"), false
	}

	if !(utils.StringInSlice(config.Sidecar, Cfg.AvailableSidecars) || config.Sidecar == "") {
		return fmt.Errorf("Sidecar provided is not an available sidecar."), false
	}

	// Check if the config type is static here and if it is
	// then return an indication for that, so that caller knows if it need
	// to check a valid environment or not.
	challengeTypes := GetAvailableChallengeTypes()
	for i := range challengeTypes {
		if challengeTypes[i] == config.Type {
			if config.Type == core.STATIC_CHALLENGE_TYPE_NAME {
				// Challenge is a standalone static challenge
				// No need to validate environment, since we don't need that.
				return nil, true
			}

			return nil, false
		}
	}

	return fmt.Errorf("Not a valid challenge type : %s", config.Type), false
}

// This contains challenge specific properties which includes
//
// * AptDeps: Apt dependencies for the challenge
// * SetupScripts: relative path to the challenge setup scripts
// * StaticContentDir: Relative path to the directory which you want
// 		to serve statically for the challenge, for example a libc for binary
// 		challenge.
// * RunCmd: Command to run or start the challenge.
// * Base for the challenge, this might be extension to dockerfile usage
// 		like for a php challenge this can be php:web, for node node:web
// 		for xinetd services xinetd:service
// * Ports: A list of ports to be used by the challenge.
// * WebRoot: relative path to web challenge directory
// * DefaultPort: default port for application
type ChallengeEnv struct {
	AptDeps          []string         `toml:"apt_deps"`
	Ports            []uint32         `toml:"ports"`
	SetupScripts     []string         `toml:"setup_scripts"`
	StaticContentDir string           `toml:"static_dir"`
	RunCmd           string           `toml:"run_cmd"`
	Base             string           `toml:"base"`
	BaseImage        string           `toml:"base_image"`
	WebRoot          string           `toml:"web_root"`
	DefaultPort      uint32           `toml:"default_port"`
	ServicePath      string           `toml:"service_path"`
	EnvironmentVars  []EnvironmentVar `toml:"var"`
}

func (config *ChallengeEnv) ValidateRequiredFields(challType string, challdir string) error {
	if len(config.Ports) == 0 {
		return errors.New("Are you sure you have specified the ports used by challenge")
	}

	if len(config.Ports) > int(core.MAX_PORT_PER_CHALL) {
		return fmt.Errorf("Max ports allowed for challenge : %d given : %d", core.MAX_PORT_PER_CHALL, len(config.Ports))
	}

	// By default if no port is specified to be default, the first port
	// from the list is assumed to be default and the service is deployed accordingly.
	if config.DefaultPort == 0 {
		config.DefaultPort = config.Ports[0]
	}

	if !utils.UInt32InList(config.DefaultPort, config.Ports) {
		return fmt.Errorf("`default_port` must be one of the Ports in the `ports` list")
	}

	for _, port := range config.Ports {
		if port < core.ALLOWED_MIN_PORT_VALUE || port > core.ALLOWED_MAX_PORT_VALUE {
			return fmt.Errorf("Port value must be between %s and %s", core.ALLOWED_MIN_PORT_VALUE, core.ALLOWED_MAX_PORT_VALUE)
		}
	}

	if config.StaticContentDir != "" {
		if filepath.IsAbs(config.StaticContentDir) {
			return fmt.Errorf("Static content directory path should be relative to challenge directory root")
		}
		if err := utils.ValidateDirExists(filepath.Join(challdir, config.StaticContentDir)); err != nil {
			return err
		}
	}

	// Run command is only a required value in case of bare challenge types.
	if config.RunCmd == "" && challType == core.BARE_CHALLENGE_TYPE_NAME {
		return fmt.Errorf("A valid run_cmd should be provided for the challenge environment")
	}

	if config.BaseImage == "" {
		config.BaseImage = core.DEFAULT_BASE_IMAGE
	}

	if !utils.StringInSlice(config.BaseImage, Cfg.AllowedBaseImages) {
		return fmt.Errorf("The base image: %s is not supported", config.BaseImage)
	}

	if challType == core.SERVICE_CHALLENGE_TYPE_NAME {
		// Challenge type is service.
		// ServicePath must be relative.
		if config.ServicePath != "" {
			if filepath.IsAbs(config.ServicePath) {
				return fmt.Errorf("For challenge type `services` service_path is a required variable, which should be relative path to executable.")
			} else if err := utils.ValidateFileExists(filepath.Join(challdir, config.ServicePath)); err != nil {
				return fmt.Errorf("File %s does not exist", config.ServicePath)
			}
		}
	} else if strings.HasPrefix(challType, "web") {
		// Challenge type is web.
		if config.WebRoot == "" {
			return errors.New("Web root can not be empty for web challenges")
		} else if config.WebRoot != "" {
			if filepath.IsAbs(config.WebRoot) {
				return fmt.Errorf("Web Root directory path should be relative to challenge directory root")
			} else if err := utils.ValidateDirExists(filepath.Join(challdir, config.WebRoot)); err != nil {
				return fmt.Errorf("Web Root directory does not exist")
			}
		}
	}

	for _, script := range config.SetupScripts {
		if filepath.IsAbs(script) {
			return fmt.Errorf("script path is absolute : %s", script)
		} else if err := utils.ValidateFileExists(filepath.Join(challdir, script)); err != nil {
			return fmt.Errorf("File %s does not exist", script)
		}
	}

	for _, env := range config.EnvironmentVars {
		if filepath.IsAbs(env.Value) {
			return fmt.Errorf("Environment Variable contains contains absolute path : %s", env.Value)
		} else if err := utils.ValidateFileExists(filepath.Join(challdir, env.Value)); err != nil {
			return fmt.Errorf("File %s does not exist", env.Value)
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

type EnvironmentVar struct {
	Key   string `toml:"key"`
	Value string `toml:"value"`
}

// Resource limitations to the container runtime
//
// * Name - Name of the author of the challenge
// * Email - Email of the author
// * SSHKey - Public SSH key for the challenge author, to give the access
//		to the challenge container.
type Resources struct {
	CPUShares int64 `toml:"cpu_shares"`
	Memory    int64 `toml:"memory_limit"`
	PidsLimit int64 `toml:"pids_limit"`
}

func (config *Resources) ValidateRequiredFields() {

	if config.CPUShares <= 0 {
		log.Warn("CPU shares not provided")
		config.CPUShares = Cfg.CPUShares
	}

	if config.Memory <= 0 {
		log.Warn("Memory Limit not provided")
		config.Memory = Cfg.Memory
	}

	if config.PidsLimit <= 0 {
		log.Warn("Pids Limit not provided")
		config.PidsLimit = Cfg.PidsLimit
	}

}
