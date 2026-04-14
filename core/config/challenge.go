package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/pkg/cr"
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
	Challenge   Challenge `toml:"challenge"`
	Author      Author    `toml:"author"`
	Resources   Resources `toml:"resource"`
	Maintainers []Author  `toml:"maintainer"`
}

func (config *BeastChallengeConfig) PopulateDefaultValues() {
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
	Metadata.DynamicFlag = false
	Metadata.Flag = "ChallengeFlag"
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

	for _, maintainer := range config.Maintainers {
		err = maintainer.ValidateRequiredFields()
		if err != nil {
			log.Debugf("Error while validating `Maintainer`'s required fields : %s", err.Error())
			return err
		}
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
// ```
type ChallengeMetadata struct {
	Flag        string   `toml:"flag"`
	Name        string   `toml:"name"`
	Type        string   `toml:"type"`
	Tags        []string `toml:"tags"`
	Description string   `toml:"description"`
	Hints       []struct {
		Text   string `toml:"text"`
		Points uint   `toml:"points"`
	} `toml:"hints"`
	MaxAttemptLimit    int      `toml:"maxAttemptLimit"`
	PreReqs            []string `toml:"preReqs"`
	DynamicFlag        bool     `toml:"dynamicFlag"`
	Points             uint     `toml:"points"`
	MaxPoints          uint     `toml:"maxPoints"`
	MinPoints          uint     `toml:"minPoints"`
	Assets             []string `toml:"assets"`
	AdditionalLinks    []string `toml:"additionalLinks"`
	Difficulty         string   `toml:"difficulty"`
	Instanced          bool     `toml:"instanced"`
	InstanceExpiration int64    `toml:"instance_expiration"`
}

func (config *ChallengeMetadata) IsInstanced() bool {
	return config.Instanced
}

func (config *ChallengeMetadata) GetInstanceExpiration() int64 {
	if config.InstanceExpiration > 0 {
		return config.InstanceExpiration
	}
	if Cfg != nil && Cfg.InstanceConfig.DefaultExpiration > 0 {
		return Cfg.InstanceConfig.DefaultExpiration
	}
	return 300
}

// In this validation returned boolean value represents if the challenge type is
// static or not.
func (config *ChallengeMetadata) ValidateRequiredFields() (error, bool) {
	if config.Name == "" || (config.Flag == "" && !config.DynamicFlag) {
		return fmt.Errorf("name and flag required for the challenge"), false
	}

	// Checks if fail solve limit is provided and is greater than 0
	if config.MaxAttemptLimit < 0 {
		return fmt.Errorf("fail solve limit must be greater than equal to 0"), false
	} else if config.MaxAttemptLimit == 0 {
		// sets default value to -1 so it means that there is no limit.
		log.Warn("MaxAttemptLimit is set to 0, defaulting to no limit for attempts")
		config.MaxAttemptLimit = -1
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

	return fmt.Errorf("not a valid challenge type : %s", config.Type), false
}

// This contains challenge specific properties which includes the following toml fields
//
// ```toml
// # Ports to reserve for the challenge, we bind only one of these to host other are for internal communictaions only.
// # Should be within a particular permissible range.
// ports = [0, 0]
// default_port = 0 # Default port to use for any port specific action by beast. This is the container port.
//
// # Ports can also be specified as a mapping between host and the container.
// # This can be used when we need customized port mapping between container and the host.
// port_mappings = ["10001:80"]
//
// # Dependencies required by challenge, installed using default package manager of base image apt for most cases.
// apt_deps = ["", ""]
//
// # A list of setup scripts to run for building challenge enviroment.
// # Keep in mind that these are only for building the challenge environment and are executed
// # in the iamge building step of the deployment pipeline.
// setup_scripts = ["", ""]
//
// # A directory containing any of the static assets for the challenge, exposed by beast static endpoint.
// static_dir = ""
//
// # Command to execute inside the container, if a predefined type is being used try to
// # use an existing field to let beast automatically calculate what command to run.
// # If you want to host a binary using xinetd use type service and specify absolute path
// # of the service using service_path field.
// run_cmd = ""
//
// # Similar to run_cmd but in this case you have the entire container to yourself
// # and everything you are doing is done using root permissions inside the container
// # When using this keep in mind you are root inside the container.
// entrypoint = ""
//
// # Relative path to binary which needs to be executed when the specified
// # Type for the challenge is service.
// # This can be anything which can be exeucted, a python file, a binary etc.
// service_path = ""
//
// # Relative directory corresponding to root of the challenge where the root
// # of the web application lies.
// web_root = ""
//
// # Any custom base image you might want to use for your particular challenge.
// # Exists for flexibility reasons try to use existing base iamges wherever possible.
// base_image = ""
//
// # Docker file name for specific type challenge - `docker`.
// # Helps to build flexible images for specific user-custom challenges
// docket_context = ""
//
// # Environment variables that can be used in the application code.
// [[var]]
//
//	key = ""
//	value = ""
//
// [[var]]
//
//	key = ""
//	value = ""
//
// Type of traffic to expose through the port mapping provided.
// traffic = "udp" / "tcp"
// ```
type ChallengeEnv struct {
	AptDeps          []string         `toml:"apt_deps"`
	Ports            []uint32         `toml:"ports"`
	DefaultPort      uint32           `toml:"default_port"`
	PortVariables    []string         `toml:"-"`
	DefaultPortVar   string           `toml:"default_port_var"`
	SetupScripts     []string         `toml:"setup_scripts"`
	StaticContentDir string           `toml:"static_dir"`
	RunCmd           string           `toml:"run_cmd"`
	BaseImage        string           `toml:"base_image"`
	WebRoot          string           `toml:"web_root"`
	ServicePath      string           `toml:"service_path"`
	Entrypoint       string           `toml:"entrypoint"`
	DockerCtx        string           `toml:"docker_context"`
	XinetdConf       string           `toml:"xinetd_conf"`
	DockerCompose    string           `toml:"docker_compose"`
	EnvironmentVars  []EnvironmentVar `toml:"var"`
	Traffic          string           `toml:"traffic"`
}

func (config *ChallengeEnv) TrafficType() cr.TrafficType {
	if config.Traffic == "" {
		return cr.DefaultTraffic
	}
	return cr.TrafficType(config.Traffic)
}

// GetDefaultPort returns the default port used by the challenge from the challenge environment
// configuration.
func (config *ChallengeEnv) GetDefaultPort() uint32 {
	ports := config.Ports
	if len(ports) == 0 {
		return 0
	}

	return ports[0]
}

// ValidateRequiredFields validates required fields for the Challenge environment configuration.
// This requires challenge type to be passed so that we can verfiy based on type
// of the challenge.
func (config *ChallengeEnv) ValidateRequiredFields(challType string, challdir string) error {
	// Validate port related stuff for the challenge environment configuration.

	if config.StaticContentDir != "" {
		if filepath.IsAbs(config.StaticContentDir) {
			return fmt.Errorf("static content directory path should be relative to challenge directory root")
		}
		if err := utils.ValidateDirExists(filepath.Join(challdir, config.StaticContentDir)); err != nil {
			return err
		}
	}

	if config.Entrypoint != "" && config.RunCmd != "" {
		return fmt.Errorf("run_cmd cannot be non empty when entrypoint is provided")
	}

	if config.DockerCompose != "" {
		if filepath.IsAbs(config.DockerCompose) {
			return fmt.Errorf("docker_compose path should be relative to challenge directory root")
		}
		if err := utils.ValidateFileExists(filepath.Join(challdir, config.DockerCompose)); err != nil {
			return fmt.Errorf("docker_compose file does not exist: %s", config.DockerCompose)
		}

		// Warn if other configuration fields are specified when docker_compose is provided
		if config.RunCmd != "" {
			log.Warn("run_cmd will be ignored when docker_compose is specified")
		}
		if config.Entrypoint != "" {
			log.Warn("entrypoint will be ignored when docker_compose is specified")
		}
		if config.DockerCtx != "" {
			log.Warn("docker_context will be ignored when docker_compose is specified")
		}
		if config.BaseImage != core.DEFAULT_BASE_IMAGE {
			log.Warn("base_image will be ignored when docker_compose is specified")
		}
		if config.WebRoot != "" {
			log.Warn("web_root will be ignored when docker_compose is specified")
		}
		if config.ServicePath != "" {
			log.Warn("service_path will be ignored when docker_compose is specified")
		}
		if len(config.AptDeps) > 0 {
			log.Warn("apt_deps will be ignored when docker_compose is specified")
		}
		if len(config.SetupScripts) > 0 {
			log.Warn("setup_scripts will be ignored when docker_compose is specified")
		}

		if err := config.ExtractPortsCompose(challdir); err != nil {
			return err
		}
		return nil
	}

	if err := config.ExtractPorts(); err != nil {
		return err
	}

	// Run command is only a required value in case of bare challenge types.
	if config.RunCmd == "" && config.Entrypoint == "" && config.DockerCtx == "" && config.DockerCompose == "" && challType == core.BARE_CHALLENGE_TYPE_NAME {
		return fmt.Errorf("a valid run_cmd should be provided for the challenge environment")
	}

	if config.BaseImage == "" {
		config.BaseImage = core.DEFAULT_BASE_IMAGE
	}

	if !utils.StringInSlice(config.BaseImage, Cfg.AllowedBaseImages) {
		return fmt.Errorf("the base image: %s is not supported", config.BaseImage)
	}

	if challType == core.SERVICE_CHALLENGE_TYPE_NAME {
		// Challenge type is service.
		// ServicePath must be relative.
		if config.ServicePath != "" {
			if filepath.IsAbs(config.ServicePath) {
				return fmt.Errorf("for challenge type `services` service_path is a required variable, which should be relative path to executable")
			} else if err := utils.ValidateFileExists(filepath.Join(challdir, config.ServicePath)); err != nil {
				// Skip this, we might create service later too.
				log.Warnf("Service path file %s does not exist", config.ServicePath)
			}
		}
	} else if strings.HasPrefix(challType, core.WEB_CHALLENGE_TYPE_NAME) {
		// Challenge type is web.
		if config.WebRoot == "" && config.DockerCtx == "" && config.DockerCompose == "" {
			return errors.New("web root can not be empty for web challenges without custom dockerfile or docker-compose")
		} else if config.WebRoot != "" {
			if filepath.IsAbs(config.WebRoot) {
				return fmt.Errorf("web Root directory path should be relative to challenge directory root")
			} else if err := utils.ValidateDirExists(filepath.Join(challdir, config.WebRoot)); err != nil {
				return fmt.Errorf("web Root directory does not exist")
			}
		}
	}

	for _, script := range config.SetupScripts {
		if filepath.IsAbs(script) {
			return fmt.Errorf("script path is absolute : %s", script)
		} else if err := utils.ValidateFileExists(filepath.Join(challdir, script)); err != nil {
			return fmt.Errorf("file %s does not exist", script)
		}
	}

	for _, env := range config.EnvironmentVars {
		if filepath.IsAbs(env.Value) {
			return fmt.Errorf("environment Variable contains absolute path : %s", env.Value)
		} else if err := utils.ValidateFileExists(filepath.Join(challdir, env.Value)); err != nil {
			return fmt.Errorf("file %s does not exist", env.Value)
		}
	}

	if config.Entrypoint != "" {
		if filepath.IsAbs(config.Entrypoint) {
			return fmt.Errorf("entrypoint contains absolute path : %s", config.Entrypoint)
		} else if err := utils.ValidateFileExists(filepath.Join(challdir, config.Entrypoint)); err != nil {
			return fmt.Errorf("file %s does not exist", config.Entrypoint)
		}
	}

	if config.Traffic != "" && !cr.IsValidTrafficType(config.Traffic) {
		return fmt.Errorf("not a valid traffic type provided, required (%v), got %s", cr.GetValidTrafficTypes(), config.Traffic)
	}

	return nil
}

func (config *ChallengeEnv) ExtractPorts() error {
	if config.DockerCompose == "" {
		if len(config.Ports) == 0 && config.DefaultPort == 0 {
			return errors.New("some port is required to be specified by the challenge")
		}
		if len(config.Ports) > int(core.MAX_PORT_PER_CHALL) {
			return fmt.Errorf("max ports allowed for challenge : %d given : %d", core.MAX_PORT_PER_CHALL, len(config.Ports))
		}

		if config.DefaultPort == 0 {
			config.DefaultPort = config.Ports[0]
			log.Warnf("default port is 0 for challenge with default port : %d", config.Ports[0])
		} else if !utils.UInt32InList(config.DefaultPort, config.Ports) {
			return fmt.Errorf("default port %d was not found in assigned ports", config.DefaultPort)
		}
	}

	return nil
}

func (config *ChallengeEnv) ExtractPortsCompose(challdir string) error {
	if config.DockerCompose != "" {
		portVariables, err := utils.ExtractPortsFromCompose(filepath.Join(challdir, config.DockerCompose))
		if err != nil {
			log.Warnf("failed to extract port variables from compose file with the following error : %s", err.Error())
		}
		if len(portVariables) == 0 {
			return errors.New("some port is required to be specified by the challenge")
		}

		config.PortVariables = portVariables
		if config.DefaultPortVar == "" {
			config.DefaultPortVar = config.PortVariables[0]
			log.Warnf("default port variable is empty, settting it to %s", config.PortVariables[0])
		}
		if !utils.StringInSlice(config.DefaultPortVar, config.PortVariables) {
			return fmt.Errorf("default port variable: %s was not found", config.DefaultPortVar)
		}
	}

	return nil
}

// Metadata related to author of the challenge, this structure includes
//
//   - Name - Name of the author of the challenge
//   - Email - Email of the author
//   - SSHKey - Public SSH key for the challenge author, to give the access
//     to the challenge container.
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
	CPUShares int64   `toml:"cpu_shares"`
	Memory    int64   `toml:"memory_limit"`
	PidsLimit int64   `toml:"pids_limit"`
	CPUsLimit float32 `toml:"cpuslimit"`
}

func (config *Resources) ValidateRequiredFields() {
	if config.CPUShares <= 0 {
		log.Debug("CPU shares not provided in configuration, using default.")
		config.CPUShares = Cfg.CPUShares
	}

	if config.Memory <= 0 {
		log.Debug("Memory Limit not provided in configuration, using default.")
		config.Memory = Cfg.Memory
	}

	if config.PidsLimit <= 0 {
		log.Debug("Pids Limit not provided in configuration, using default.")
		config.PidsLimit = Cfg.PidsLimit
	}

	if config.CPUsLimit <= 0 {
		log.Debug("CPUsLimit not provided in configuration, using default.")
		config.CPUsLimit = Cfg.CPUsLimit
	}
}
