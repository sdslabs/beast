package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

const SERVICE_CONTAINER_DEPS string = "xinetd"
const SERVICE_CHALL_RUN_CMD string = "xinetd -dontfork"

var challengeNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var environmentKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var generatedPathPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,255}$`)

func IsValidChallengeName(name string) bool {
	return challengeNamePattern.MatchString(name)
}

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
	Author.Name = "Author Name"
	Author.Email = "author@example.com"
}

func (Metadata *ChallengeMetadata) PopulateChallengeMetadata() {
	Metadata.Name = "challenge-name"
	Metadata.Type = core.STATIC_CHALLENGE_TYPE_NAME
	Metadata.DynamicFlag = false
	Metadata.Flag = "flag{replace-me}"
	Metadata.Difficulty = "medium"
	Metadata.Points = 100
}

func (Env *ChallengeEnv) PopulateChallengeEnv() {
	Env.AptDeps = []string{}
	Env.Ports = []uint32{}
	Env.SetupScripts = []string{}
	Env.StaticContentDir = core.PUBLIC
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

	if err = config.Resources.ValidateRequiredFields(); err != nil {
		return err
	}
	if config.Challenge.Env.DockerCompose != "" {
		composePath, err := utils.ResolvePathWithin(challdir, config.Challenge.Env.DockerCompose)
		if err != nil {
			return err
		}
		if err := utils.ValidateComposeResources(composePath, config.Resources.Memory, config.Resources.PidsLimit, config.Resources.CPUsLimit); err != nil {
			return fmt.Errorf("validate Compose resources: %w", err)
		}
	}

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
		if config.Env.StaticContentDir == "" {
			config.Env.StaticContentDir = core.PUBLIC
		}
		if err := validateChallengeDir(challdir, config.Env.StaticContentDir, "static_dir"); err != nil {
			return err
		}
		return config.Metadata.ValidateAssets(challdir, config.Env.StaticContentDir)
	}

	err = config.Env.ValidateRequiredFields(config.Metadata.Type, challdir)
	if err != nil {
		log.Debugf("Error while validating `ChallengeEnv`'s required fields : %s", err.Error())
		return err
	}
	return config.Metadata.ValidateAssets(challdir, config.Env.StaticContentDir)
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
	if !IsValidChallengeName(config.Name) {
		return fmt.Errorf("challenge name must match %s", challengeNamePattern.String()), false
	}
	if config.MaxPoints > 0 && config.MinPoints > config.MaxPoints {
		return fmt.Errorf("minPoints cannot exceed maxPoints"), false
	}
	if config.MaxPoints == 0 && config.Points > 0 && config.MinPoints > config.Points {
		return fmt.Errorf("minPoints cannot exceed points when maxPoints is omitted"), false
	}
	if config.MaxPoints > 0 && config.Points > config.MaxPoints {
		return fmt.Errorf("points cannot exceed maxPoints"), false
	}
	for _, prerequisite := range config.PreReqs {
		if !challengeNamePattern.MatchString(prerequisite) {
			return fmt.Errorf("invalid prerequisite challenge name %q", prerequisite), false
		}
	}
	for _, link := range config.AdditionalLinks {
		parsed, err := url.ParseRequestURI(link)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			return fmt.Errorf("invalid additional link %q", link), false
		}
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

func (config *ChallengeMetadata) ValidateAssets(challengeDir, staticContentDir string) error {
	if len(config.Assets) == 0 {
		return nil
	}
	if staticContentDir == "" {
		staticContentDir = core.PUBLIC
	}
	staticRoot, err := utils.ResolvePathWithin(challengeDir, staticContentDir)
	if err != nil {
		return fmt.Errorf("invalid static asset root: %w", err)
	}
	for _, asset := range config.Assets {
		assetPath, err := utils.ResolvePathWithin(staticRoot, asset)
		if err != nil {
			return fmt.Errorf("invalid challenge asset %q: %w", asset, err)
		}
		if err := utils.ValidateFileExists(assetPath); err != nil {
			return fmt.Errorf("invalid challenge asset %q: %w", asset, err)
		}
	}
	return nil
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
	if config.DefaultPort != 0 {
		return config.DefaultPort
	}
	ports := config.Ports
	if len(ports) == 0 {
		return 0
	}

	return ports[0]
}

func validateChallengeFile(challengeDir, relativePath, field string) error {
	resolvedPath, err := utils.ResolvePathWithin(challengeDir, relativePath)
	if err != nil {
		return fmt.Errorf("invalid %s %q: %w", field, relativePath, err)
	}
	if err := utils.ValidateFileExists(resolvedPath); err != nil {
		return fmt.Errorf("invalid %s %q: %w", field, relativePath, err)
	}
	return nil
}

func validateChallengeDir(challengeDir, relativePath, field string) error {
	resolvedPath, err := utils.ResolvePathWithin(challengeDir, relativePath)
	if err != nil {
		return fmt.Errorf("invalid %s %q: %w", field, relativePath, err)
	}
	if err := utils.ValidateDirExists(resolvedPath); err != nil {
		return fmt.Errorf("invalid %s %q: %w", field, relativePath, err)
	}
	return nil
}

func validateGeneratedChallengeFile(challengeDir, relativePath, field string, setupScripts []string) error {
	cleaned := filepath.ToSlash(filepath.Clean(relativePath))
	if !generatedPathPattern.MatchString(cleaned) || cleaned == "." || strings.Contains(cleaned, "../") {
		return fmt.Errorf("invalid %s path %q", field, relativePath)
	}
	if err := validateChallengeFile(challengeDir, cleaned, field); err == nil {
		return nil
	} else if _, statErr := os.Lstat(filepath.Join(challengeDir, filepath.FromSlash(cleaned))); !os.IsNotExist(statErr) {
		return err
	}
	if len(setupScripts) == 0 {
		return fmt.Errorf("%s %q does not exist and no setup script generates it", field, relativePath)
	}
	return nil
}

// ValidateRequiredFields validates required fields for the Challenge environment configuration.
// This requires challenge type to be passed so that we can verfiy based on type
// of the challenge.
func (config *ChallengeEnv) ValidateRequiredFields(challType string, challdir string) error {
	// Validate port related stuff for the challenge environment configuration.

	if config.StaticContentDir != "" {
		if err := validateChallengeDir(challdir, config.StaticContentDir, "static_dir"); err != nil {
			return err
		}
	}

	if config.Entrypoint != "" && config.RunCmd != "" {
		return fmt.Errorf("run_cmd cannot be non empty when entrypoint is provided")
	}

	if config.DockerCompose != "" {
		if err := validateChallengeFile(challdir, config.DockerCompose, "docker_compose"); err != nil {
			return err
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
	if config.DockerCtx != "" {
		if err := validateChallengeFile(challdir, config.DockerCtx, "docker_context"); err != nil {
			return err
		}
	}
	if config.XinetdConf != "" {
		if err := validateChallengeFile(challdir, config.XinetdConf, "xinetd_conf"); err != nil {
			return err
		}
	}

	if challType == core.SERVICE_CHALLENGE_TYPE_NAME {
		// Challenge type is service.
		// ServicePath must be relative.
		if config.ServicePath != "" {
			if err := validateGeneratedChallengeFile(challdir, config.ServicePath, "service_path", config.SetupScripts); err != nil {
				return err
			}
		}
	} else if strings.HasPrefix(challType, core.WEB_CHALLENGE_TYPE_NAME) {
		// Challenge type is web.
		if config.WebRoot == "" && config.DockerCtx == "" && config.DockerCompose == "" {
			return errors.New("web root can not be empty for web challenges without custom dockerfile or docker-compose")
		} else if config.WebRoot != "" {
			if err := validateChallengeDir(challdir, config.WebRoot, "web_root"); err != nil {
				return err
			}
		}
	}

	for _, script := range config.SetupScripts {
		if err := validateChallengeFile(challdir, script, "setup_scripts"); err != nil {
			return err
		}
	}

	for _, env := range config.EnvironmentVars {
		if !environmentKeyPattern.MatchString(env.Key) {
			return fmt.Errorf("invalid environment variable key %q", env.Key)
		}
		if err := validateChallengeFile(challdir, env.Value, "environment variable value"); err != nil {
			return err
		}
	}

	if config.Entrypoint != "" {
		if err := validateChallengeFile(challdir, config.Entrypoint, "entrypoint"); err != nil {
			return err
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
		seen := make(map[uint32]bool, len(config.Ports))
		for _, port := range config.Ports {
			if port == 0 || port > 65535 {
				return fmt.Errorf("container port %d is outside 1-65535", port)
			}
			if seen[port] {
				return fmt.Errorf("container port %d is duplicated", port)
			}
			seen[port] = true
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
		composePath, err := utils.ResolvePathWithin(challdir, config.DockerCompose)
		if err != nil {
			return err
		}
		portVariables, err := utils.ExtractPortsFromCompose(composePath)
		if err != nil {
			return fmt.Errorf("extract compose ports: %w", err)
		}
		if len(portVariables) == 0 {
			return errors.New("some port is required to be specified by the challenge")
		}
		if len(portVariables) > int(core.MAX_PORT_PER_CHALL) {
			return fmt.Errorf("max ports allowed for challenge: %d given: %d", core.MAX_PORT_PER_CHALL, len(portVariables))
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
//
// ```toml
// # Optional fields
// name = ""
//
// # Required Fields
// email = ""
// ```
type Author struct {
	Name  string `toml:"name"`
	Email string `toml:"email"`
}

func (config *Author) ValidateRequiredFields() error {
	if config.Email == "" {
		return errors.New("challenge author email is required")
	}
	address, err := mail.ParseAddress(config.Email)
	if err != nil || address.Address != config.Email {
		return fmt.Errorf("invalid challenge author email %q", config.Email)
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

func (config *Resources) ValidateRequiredFields() error {
	if Cfg == nil {
		return errors.New("global configuration is not initialized")
	}
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
	if err := cr.ValidateResourceLimits(config.CPUShares, config.CPUsLimit, config.Memory, config.PidsLimit); err != nil {
		return fmt.Errorf("invalid challenge resource limits: %w", err)
	}
	if config.CPUShares > Cfg.CPUShares {
		return fmt.Errorf("cpu_shares %d exceeds global limit %d", config.CPUShares, Cfg.CPUShares)
	}
	if config.Memory > Cfg.Memory {
		return fmt.Errorf("memory_limit %d exceeds global limit %d", config.Memory, Cfg.Memory)
	}
	if config.PidsLimit > Cfg.PidsLimit {
		return fmt.Errorf("pids_limit %d exceeds global limit %d", config.PidsLimit, Cfg.PidsLimit)
	}
	if config.CPUsLimit > Cfg.CPUsLimit {
		return fmt.Errorf("cpuslimit %.2f exceeds global limit %.2f", config.CPUsLimit, Cfg.CPUsLimit)
	}
	return nil
}
