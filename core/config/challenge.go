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
// sidecar = "" # Name of the sidecar if any used by the challenge.
// ```
type ChallengeMetadata struct {
	Flag            string   `toml:"flag"`
	Name            string   `toml:"name"`
	Type            string   `toml:"type"`
	Tags            []string `toml:"tags"`
	Sidecar         string   `toml:"sidecar"`
	Description     string   `toml:"description"`
	Hints           []string `toml:"hints"`
	Points          uint     `toml:"points"`
	Assets          []string `toml:"assets"`
	AdditionalLinks []string `toml:"additionalLinks"`
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
// default_port = 0 # Default port to use for any port specific action by beast. This is the container port.
//
// # Ports can also be specified as a mapping between host and the container.
// # This can be used when we need customized port mapping between container and the host.
// port_mappings = ["10001:80"]
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
// # Docker file name for specific type challenge - `docker`.
// # Helps to build flexible images for specific user-custom challenges
// docket_context = ""
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
//
// Type of traffic to expose through the port mapping provided.
// traffic = "udp" / "tcp"
// ```
type ChallengeEnv struct {
	AptDeps          []string         `toml:"apt_deps"`
	Ports            []uint32         `toml:"ports"`
	DefaultPort      uint32           `toml:"default_port"`
	PortMappings     []string         `toml:"port_mappings"`
	SetupScripts     []string         `toml:"setup_scripts"`
	StaticContentDir string           `toml:"static_dir"`
	RunCmd           string           `toml:"run_cmd"`
	BaseImage        string           `toml:"base_image"`
	WebRoot          string           `toml:"web_root"`
	ServicePath      string           `toml:"service_path"`
	Entrypoint       string           `toml:"entrypoint"`
	DockerCtx        string           `toml:"docker_context"`
	EnvironmentVars  []EnvironmentVar `toml:"var"`
	Traffic          string           `toml:"traffic"`
}

func (config *ChallengeEnv) TrafficType() cr.TrafficType {
	if config.Traffic == "" {
		return cr.DefaultTraffic
	}
	return cr.TrafficType(config.Traffic)
}

// NewPortMapping returns a new port mapping instance.
func NewPortMapping(hp, cp uint32) cr.PortMapping {
	return cr.PortMapping{
		HostPort:      hp,
		ContainerPort: cp,
	}
}

// Given a port mapping array and a port the function checks whether the port exists in the mapping
// as a container port.
func checkIfPortExistInMapping(portMapping []cr.PortMapping, port uint32) bool {
	for _, portMap := range portMapping {
		if port == portMap.ContainerPort {
			return true
		}
	}

	return false
}

// GetPortMappings returns the entire port mapping for the challenge from the challenge
// environment configuration.
func (config *ChallengeEnv) GetPortMappings() ([]cr.PortMapping, error) {
	var mapping []cr.PortMapping

	var containerPorts []uint32
	for _, portMap := range config.PortMappings {
		hp, cp, err := utils.ParsePortMapping(portMap)
		if err != nil {
			return mapping, err
		}
		mapping = append(mapping, NewPortMapping(hp, cp))
		containerPorts = append(containerPorts, cp)
	}

	for _, port := range config.Ports {
		if !utils.UInt32InList(port, containerPorts) {
			containerPorts = append(containerPorts, port)
			mapping = append(mapping, NewPortMapping(port, port))
		}
	}

	return mapping, nil
}

// GetAllHostPorts is utility function for the ChallengeEnv configuration which returns
// the entire list of all the host ports which are being used by the challenge.
func (config *ChallengeEnv) GetAllHostPorts() ([]uint32, error) {
	var hostPorts []uint32
	var containerPorts []uint32

	for _, portMap := range config.PortMappings {
		hp, cp, err := utils.ParsePortMapping(portMap)
		if err != nil {
			return hostPorts, err
		}
		hostPorts = append(hostPorts, hp)
		containerPorts = append(containerPorts, cp)
	}

	for _, port := range config.Ports {
		if !utils.UInt32InList(port, containerPorts) {
			hostPorts = append(hostPorts, port)
			containerPorts = append(containerPorts, port)
		}
	}

	return hostPorts, nil
}

// GetAllContainerPorts is utility function for the ChallengeEnv configuration which returns
// the entire list of all the container ports which are being used by the challenge.
func (config *ChallengeEnv) GetAllContainerPorts() ([]uint32, error) {
	var containerPorts []uint32

	for _, portMap := range config.PortMappings {
		_, cp, err := utils.ParsePortMapping(portMap)
		if err != nil {
			return containerPorts, err
		}
		containerPorts = append(containerPorts, cp)
	}

	for _, port := range config.Ports {
		if !utils.UInt32InList(port, containerPorts) {
			containerPorts = append(containerPorts, port)
		}
	}

	return containerPorts, nil
}

// GetDefaultPort returns the default port used by the challenge from the challenge environment
// configuration.
func (config *ChallengeEnv) GetDefaultPort() uint32 {
	mappings, err := config.GetPortMappings()
	if err != nil || len(mappings) == 0 {
		return 0
	}

	return mappings[0].ContainerPort
}

// ValidateRequiredFields validates required fields for the Challenge environment configuration.
// This requires challenge type to be passed so that we can verfiy based on type
// of the challenge.
func (config *ChallengeEnv) ValidateRequiredFields(challType string, challdir string) error {
	// Validate port related stuff for the challenge environment configuration.
	if len(config.Ports) == 0 && len(config.PortMappings) == 0 {
		return errors.New("Some port is required to be specified by the challenge")
	}

	if len(config.Ports)+len(config.PortMappings) > int(core.MAX_PORT_PER_CHALL) {
		return fmt.Errorf("Max ports allowed for challenge : %d given : %d", core.MAX_PORT_PER_CHALL, len(config.Ports))
	}

	portMappings, err := config.GetPortMappings()
	if err != nil {
		return fmt.Errorf("Error while parsing port mapping: %s", err)
	}

	// By default if no port is specified to be default, the first port
	// from the list is assumed to be default and the service is deployed accordingly.
	if config.DefaultPort == 0 {
		config.DefaultPort = portMappings[0].ContainerPort
	}

	if !checkIfPortExistInMapping(portMappings, config.DefaultPort) {
		return fmt.Errorf("`default_port` must be one of the Ports in the `ports` list")
	}

	for _, portMap := range portMappings {
		if portMap.HostPort < core.ALLOWED_MIN_PORT_VALUE || portMap.HostPort > core.ALLOWED_MAX_PORT_VALUE {
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

	if config.Traffic != "" && !cr.IsValidTrafficType(config.Traffic) {
		return fmt.Errorf("Not a valid traffic type provided, required (%v), got %s", cr.GetValidTrafficTypes(), config.Traffic)
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
}
