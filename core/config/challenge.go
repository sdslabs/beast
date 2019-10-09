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

func (config *BeastChallengeConfig) PopulateDefualtValues() {
	config.Author.PopulateAuthor()
	config.Challenge.Metadata.PopulateChallengeMetadata()
	config.Challenge.Env.PopulateChallengeEnv()
}

func (Author *Author) PopulateAuthor() {
	Author.Name = "AuthorName"
	Author.Email = "AuthorMail"
	Author.SSHKey = "AuthorPubKey"
}

func (Metadata *ChallengeMetadata) PopulateChallengeMetadata() {
	Metadata.Name = "ChallengeName"
	Metadata.Type = "ChallengeType"
	Metadata.Flag = "ChallengeFlag"
	Metadata.Sidecar = "SidecarHelper"
}

func (Env *ChallengeEnv) PopulateChallengeEnv() {
	Env.AptDeps = []string{}
	Env.Ports = []uint32{}
	Env.SetupScripts = []string{}
	Env.StaticContentDir = "StaticContentDir"
	Env.BaseImage = "ChallengeBase"
	Env.RunCmd = "RunCmd"
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
// ```toml
// # Required Fields
// flag = "" # Flag for the challenge
// name = "" # Name of the challenge
// type = "" # Type of the challenge, one of - Get available types from /api/info/types/available
// description = "" # Descritption for the challenge.
//
// # Optional fields.
// tags = ["", ""] # Tags that the challenge might belong to, used to do bulk query and handling eg. binary, misc etc.
// hints = ["", ""]
// sidecar = "" # Name of the sidecar if any used by the challenge.
// ```
type ChallengeMetadata struct {
	Flag        string   `toml:"flag"`
	Name        string   `toml:"name"`
	Type        string   `toml:"type"`
	Tags        []string `toml:"tags"`
	Sidecar     string   `toml:"sidecar"`
	Description string   `toml:"description"`
	Hints       []string `toml:"hints"`
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

// This contains challenge specific properties which includes the following toml fields
//
// ```toml
// # Ports to reserve for the challenge, we bind only one of these to host other are for internal communictaions only.
// # Should be within a particular permissible range.
// ports = [0, 0]
// default_port = 0 # Default port to use for any port specific action by beast.
//
//
// # Dependencies required by challenge, installed using default package manager of base image apt for most cases.
// apt_deps = ["", ""]
//
//
// # A list of setup scripts to run for building challenge enviroment.
// # Keep in mind that these are only for building the challenge environment and are executed
// # in the iamge building step of the deployment pipeline.
// setup_scripts = ["", ""]
//
//
// # A directory containing any of the static assets for the challenge, exposed by beast static endpoint.
// static_dir = ""
//
//
// # Command to execute inside the container, if a predefined type is being used try to
// # use an existing field to let beast automatically calculate what command to run.
// # If you want to host a binary using xinetd use type service and specify absolute path
// # of the service using service_path field.
// run_cmd = ""
//
//
// # Similar to run_cmd but in this case you have the entire container to yourself
// # and everything you are doing is done using root permissions inside the container
// # When using this keep in mind you are root inside the container.
// entrypoint = ""
//
//
// # Relative path to binary which needs to be executed when the specified
// # Type for the challenge is service.
// # This can be anything which can be exeucted, a python file, a binary etc.
// service_path = ""
//
//
// # Relative directory corresponding to root of the challenge where the root
// # of the web application lies.
// web_root = ""
//
//
// # Any custom base image you might want to use for your particular challenge.
// # Exists for flexibility reasons try to use existing base iamges wherever possible.
// base_image = ""
//
//
// # Environment variables that can be used in the application code.
// [[var]]
//     key = ""
//     value = ""
//
// [[var]]
//     key = ""
//     value = ""
// ```
type ChallengeEnv struct {
	AptDeps          []string         `toml:"apt_deps"`
	Ports            []uint32         `toml:"ports"`
	SetupScripts     []string         `toml:"setup_scripts"`
	StaticContentDir string           `toml:"static_dir"`
	RunCmd           string           `toml:"run_cmd"`
	BaseImage        string           `toml:"base_image"`
	WebRoot          string           `toml:"web_root"`
	DefaultPort      uint32           `toml:"default_port"`
	ServicePath      string           `toml:"service_path"`
	Entrypoint       string           `toml:"entrypoint"`
	DockerCtx        string           `toml:"docker_context"`
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

	if config.Entrypoint != "" && config.RunCmd != "" {
		return fmt.Errorf("run_cmd cannot be non empty when entrypoint is provided")
	}

	// Run command is only a required value in case of bare challenge types.
	if config.RunCmd == "" && config.Entrypoint == "" && challType == core.BARE_CHALLENGE_TYPE_NAME {
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
				// Skip this, we might create service later too.
				log.Warnf("Service path file %s does not exist", config.ServicePath)
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
			return fmt.Errorf("Environment Variable contains absolute path : %s", env.Value)
		} else if err := utils.ValidateFileExists(filepath.Join(challdir, env.Value)); err != nil {
			return fmt.Errorf("File %s does not exist", env.Value)
		}
	}

	if config.Entrypoint != "" {
		if filepath.IsAbs(config.Entrypoint) {
			return fmt.Errorf("Entrypoint contains absolute path : %s", config.Entrypoint)
		} else if err := utils.ValidateFileExists(filepath.Join(challdir, config.Entrypoint)); err != nil {
			return fmt.Errorf("File %s does not exist", config.Entrypoint)
		}
	}

	if challType == core.DOCKER_CHALLENGE_TYPE_NAME {
		if config.DockerCtx == "" {
			return errors.New("Docker Context file not provided in docker-type challenge")
		} else if filepath.IsAbs(config.DockerCtx) {
			return fmt.Errorf("For challenge type `docker-type` docker_context is a required variable, which should be relative path to docker context file.")
		} else if err := utils.ValidateFileExists(filepath.Join(challdir, config.DockerCtx)); err != nil {
			return fmt.Errorf("File : %s does not exist", config.DockerCtx)
		}
	} else {
		config.DockerCtx = core.DEFAULT_DOCKER_FILE
	}

	return nil
}

// Metadata related to author of the challenge, this structure includes
//
// * Name - Name of the author of the challenge
// * Email - Email of the author
// * SSHKey - Public SSH key for the challenge author, to give the access
//		to the challenge container.
//
// ```toml
// # Optional fields
// name = ""
//
// # Required Fields
// email = ""
// ssh_key = "" # Public ssh Key of the author.
// ```
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

type Resources struct {
	CPUShares int64 `toml:"cpu_shares"`
	Memory    int64 `toml:"memory_limit"`
	PidsLimit int64 `toml:"pids_limit"`
}

func (config *Resources) ValidateRequiredFields() {
	if config.CPUShares <= 0 {
		log.Warn("CPU shares not provided in configuration, using default.")
		config.CPUShares = Cfg.CPUShares
	}

	if config.Memory <= 0 {
		log.Warn("Memory Limit not provided in configuration, using default.")
		config.Memory = Cfg.Memory
	}

	if config.PidsLimit <= 0 {
		log.Warn("Pids Limit not provided in configuration, using default.")
		config.PidsLimit = Cfg.PidsLimit
	}
}
