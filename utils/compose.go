package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type composeService struct {
	Build             interface{}   `yaml:"build"`
	Ports             []string      `yaml:"ports"`
	Privileged        bool          `yaml:"privileged"`
	NetworkMode       string        `yaml:"network_mode"`
	PID               string        `yaml:"pid"`
	IPC               string        `yaml:"ipc"`
	UTS               string        `yaml:"uts"`
	UserNSMode        string        `yaml:"userns_mode"`
	Runtime           string        `yaml:"runtime"`
	CgroupParent      string        `yaml:"cgroup_parent"`
	ContainerName     string        `yaml:"container_name"`
	Volumes           []interface{} `yaml:"volumes"`
	Devices           []string      `yaml:"devices"`
	CapAdd            []string      `yaml:"cap_add"`
	CapDrop           []string      `yaml:"cap_drop"`
	SecurityOpt       []string      `yaml:"security_opt"`
	DeviceCgroupRules []string      `yaml:"device_cgroup_rules"`
	Sysctls           interface{}   `yaml:"sysctls"`
	Secrets           interface{}   `yaml:"secrets"`
	Configs           interface{}   `yaml:"configs"`
	EnvFile           interface{}   `yaml:"env_file"`
	Extends           interface{}   `yaml:"extends"`
	CredentialSpec    interface{}   `yaml:"credential_spec"`
	VolumesFrom       []string      `yaml:"volumes_from"`
	Links             []string      `yaml:"links"`
	ExternalLinks     []string      `yaml:"external_links"`
}

type composeResource struct {
	External interface{} `yaml:"external"`
}

type Compose struct {
	Services map[string]composeService  `yaml:"services"`
	Volumes  map[string]composeResource `yaml:"volumes"`
	Networks map[string]composeResource `yaml:"networks"`
	Secrets  interface{}                `yaml:"secrets"`
	Configs  interface{}                `yaml:"configs"`
}

var portRegex = regexp.MustCompile(`\$\{([^}]+)}`)
var portVariableRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

func composeResourceIsExternal(value interface{}) bool {
	switch external := value.(type) {
	case nil:
		return false
	case bool:
		return external
	default:
		return true
	}
}

func validateComposeVolume(composeDir string, value interface{}) error {
	if volume, ok := value.(string); ok {
		parts := strings.Split(volume, ":")
		if len(parts) == 1 {
			return nil
		}
		source := parts[0]
		if source == "" || strings.Contains(source, "${") {
			return fmt.Errorf("volume source %q cannot be validated", source)
		}
		if !strings.Contains(source, "/") && !strings.HasPrefix(source, ".") && !strings.HasPrefix(source, "~") {
			return nil
		}
		if filepath.IsAbs(source) || strings.HasPrefix(source, "~") {
			return fmt.Errorf("host volume source is not allowed: %s", source)
		}
		readOnly := false
		for _, option := range parts[2:] {
			if option == "ro" || option == "readonly" {
				readOnly = true
			}
		}
		if !readOnly {
			return fmt.Errorf("project bind mount %q must be read-only", source)
		}
		if _, err := ResolvePathWithin(composeDir, source); err != nil {
			return fmt.Errorf("invalid project volume source %q: %w", source, err)
		}
		return nil
	}

	encoded, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("parse volume definition: %w", err)
	}
	var volume struct {
		Type     string `yaml:"type"`
		Source   string `yaml:"source"`
		ReadOnly bool   `yaml:"read_only"`
	}
	if err := yaml.Unmarshal(encoded, &volume); err != nil {
		return fmt.Errorf("parse volume definition: %w", err)
	}
	if volume.Type == "volume" || (volume.Type == "" && volume.Source != "") {
		return nil
	}
	if volume.Type != "bind" {
		return fmt.Errorf("unsupported volume type %q", volume.Type)
	}
	if !volume.ReadOnly {
		return fmt.Errorf("project bind mount %q must be read-only", volume.Source)
	}
	if filepath.IsAbs(volume.Source) || strings.HasPrefix(volume.Source, "~") {
		return fmt.Errorf("host volume source is not allowed: %s", volume.Source)
	}
	if _, err := ResolvePathWithin(composeDir, volume.Source); err != nil {
		return fmt.Errorf("invalid project volume source %q: %w", volume.Source, err)
	}
	return nil
}

func validateComposeBuild(composeDir string, value interface{}) error {
	if value == nil {
		return nil
	}
	if contextPath, ok := value.(string); ok {
		if _, err := ResolvePathWithin(composeDir, contextPath); err != nil {
			return fmt.Errorf("invalid build context %q: %w", contextPath, err)
		}
		return nil
	}

	encoded, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("parse build definition: %w", err)
	}
	var build struct {
		Context            string      `yaml:"context"`
		Dockerfile         string      `yaml:"dockerfile"`
		AdditionalContexts interface{} `yaml:"additional_contexts"`
		Secrets            interface{} `yaml:"secrets"`
		SSH                interface{} `yaml:"ssh"`
	}
	if err := yaml.Unmarshal(encoded, &build); err != nil {
		return fmt.Errorf("parse build definition: %w", err)
	}
	if build.AdditionalContexts != nil || build.Secrets != nil || build.SSH != nil {
		return fmt.Errorf("build additional_contexts, secrets, and ssh are not allowed")
	}
	if build.Context == "" {
		build.Context = "."
	}
	if _, err := ResolvePathWithin(composeDir, build.Context); err != nil {
		return fmt.Errorf("invalid build context %q: %w", build.Context, err)
	}
	if build.Dockerfile != "" {
		dockerfilePath := filepath.Join(build.Context, build.Dockerfile)
		if _, err := ResolvePathWithin(composeDir, dockerfilePath); err != nil {
			return fmt.Errorf("invalid build dockerfile %q: %w", build.Dockerfile, err)
		}
	}
	return nil
}

func validateComposeEnvFiles(composeDir string, value interface{}) error {
	if value == nil {
		return nil
	}
	values, ok := value.([]interface{})
	if !ok {
		values = []interface{}{value}
	}
	for _, item := range values {
		path, ok := item.(string)
		if !ok {
			encoded, err := yaml.Marshal(item)
			if err != nil {
				return fmt.Errorf("parse env_file: %w", err)
			}
			var entry struct {
				Path string `yaml:"path"`
			}
			if err := yaml.Unmarshal(encoded, &entry); err != nil {
				return fmt.Errorf("parse env_file: %w", err)
			}
			path = entry.Path
		}
		if _, err := ResolvePathWithin(composeDir, path); err != nil {
			return fmt.Errorf("invalid env_file %q: %w", path, err)
		}
	}
	return nil
}

func validateComposeSecurity(composeFile string, compose Compose) error {
	if compose.Secrets != nil || compose.Configs != nil {
		return fmt.Errorf("top-level secrets and configs are not allowed")
	}
	for name, resource := range compose.Volumes {
		if composeResourceIsExternal(resource.External) {
			return fmt.Errorf("external volume %q is not allowed", name)
		}
	}
	for name, resource := range compose.Networks {
		if composeResourceIsExternal(resource.External) {
			return fmt.Errorf("external network %q is not allowed", name)
		}
	}

	composeDir := filepath.Dir(composeFile)
	for name, service := range compose.Services {
		if service.Privileged {
			return fmt.Errorf("service %q cannot be privileged", name)
		}
		if service.NetworkMode != "" && service.NetworkMode != "bridge" && service.NetworkMode != "none" {
			return fmt.Errorf("service %q uses forbidden network_mode %q", name, service.NetworkMode)
		}
		if service.PID != "" || service.IPC != "" || service.UTS != "" || service.UserNSMode != "" {
			return fmt.Errorf("service %q requests a host namespace", name)
		}
		if service.Runtime != "" || service.CgroupParent != "" || service.ContainerName != "" {
			return fmt.Errorf("service %q overrides host-managed container settings", name)
		}
		if len(service.Devices) > 0 || len(service.DeviceCgroupRules) > 0 || service.Sysctls != nil {
			return fmt.Errorf("service %q requests additional host privileges", name)
		}
		allowedCapabilities := map[string]struct{}{
			"CHOWN": {}, "DAC_OVERRIDE": {}, "FOWNER": {}, "SETGID": {}, "SETUID": {},
		}
		for _, capability := range service.CapAdd {
			if _, allowed := allowedCapabilities[strings.ToUpper(capability)]; !allowed {
				return fmt.Errorf("service %q requests forbidden capability %q", name, capability)
			}
		}
		dropsAllCapabilities := false
		for _, capability := range service.CapDrop {
			if strings.EqualFold(capability, "ALL") {
				dropsAllCapabilities = true
			}
		}
		if !dropsAllCapabilities {
			return fmt.Errorf("service %q must set cap_drop to ALL", name)
		}
		if service.Secrets != nil || service.Configs != nil {
			return fmt.Errorf("service %q uses unsupported secrets or configs", name)
		}
		if service.Extends != nil || service.CredentialSpec != nil || len(service.VolumesFrom) > 0 || len(service.Links) > 0 || len(service.ExternalLinks) > 0 {
			return fmt.Errorf("service %q references host-managed services or credentials", name)
		}
		if err := validateComposeBuild(composeDir, service.Build); err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
		if err := validateComposeEnvFiles(composeDir, service.EnvFile); err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
		noNewPrivileges := false
		for _, option := range service.SecurityOpt {
			if option == "no-new-privileges:true" {
				noNewPrivileges = true
			} else {
				return fmt.Errorf("service %q uses forbidden security_opt %q", name, option)
			}
		}
		if !noNewPrivileges {
			return fmt.Errorf("service %q must enable no-new-privileges", name)
		}
		for _, volume := range service.Volumes {
			if err := validateComposeVolume(composeDir, volume); err != nil {
				return fmt.Errorf("service %q: %w", name, err)
			}
		}
	}
	return nil
}

func ExtractPortsFromCompose(composeFile string) ([]string, error) {
	data, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("error while reading compose file: %w", err)
	}
	var raw Compose
	err = yaml.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("error while parsing compose file: %s", err.Error())
	}
	if err := validateComposeSecurity(composeFile, raw); err != nil {
		return nil, err
	}

	portVariables := make([]string, 0)
	seen := make(map[string]bool)
	for _, service := range raw.Services {
		for _, port := range service.Ports {
			matches := portRegex.FindAllStringSubmatch(port, -1)

			if len(matches) == 0 {
				return nil, fmt.Errorf("port %s is not mapped using an env variable", port)
			}

			for _, match := range matches {
				varName := match[1]

				/* Only ${PORT} is valid, ${PORT:-DEFAULT} should fail */
				if strings.Contains(varName, ":-") {
					return nil, fmt.Errorf("port variable ${%s} uses default value syntax (:-) which is not supported; use ${%s} instead",
						varName, strings.SplitN(varName, ":-", 2)[0])
				}
				if !portVariableRegex.MatchString(varName) {
					return nil, fmt.Errorf("invalid port variable name %q", varName)
				}

				if !seen[varName] {
					seen[varName] = true
					portVariables = append(portVariables, varName)
				}
			}
		}
	}

	return portVariables, nil
}
