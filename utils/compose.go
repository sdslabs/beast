package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	ComposeChallengeMountTarget = "/challenge"
)

type Compose struct {
	Services map[string]ComposeService `yaml:"services"`
	Networks map[string]ComposeNetwork `yaml:"networks"`
	Volumes  map[string]any            `yaml:"volumes"`
}

type ComposeService struct {
	Ports        []ComposePort          `yaml:"ports"`
	Networks     ComposeServiceNetworks `yaml:"networks"`
	Volumes      []ComposeVolume        `yaml:"volumes"`
	NetworkMode  string                 `yaml:"network_mode"`
	PIDMode      string                 `yaml:"pid"`
	IPCMode      string                 `yaml:"ipc"`
	UTSMode      string                 `yaml:"uts"`
	UsernsMode   string                 `yaml:"userns_mode"`
	CgroupnsMode string                 `yaml:"cgroupns_mode"`
	CgroupParent string                 `yaml:"cgroup_parent"`
	Privileged   bool                   `yaml:"privileged"`
	CapAdd       ComposeStringSet       `yaml:"cap_add"`
	SecurityOpt  ComposeStringSet       `yaml:"security_opt"`
	Devices      []string               `yaml:"devices"`
	ExtraHosts   ComposePresence        `yaml:"extra_hosts"`
	VolumesFrom  ComposePresence        `yaml:"volumes_from"`
	EnvFile      ComposePresence        `yaml:"env_file"`
	Secrets      ComposePresence        `yaml:"secrets"`
	Configs      ComposePresence        `yaml:"configs"`
	Sysctls      ComposePresence        `yaml:"sysctls"`
	Ulimits      ComposePresence        `yaml:"ulimits"`
}

type ComposeNetwork struct {
	Driver   string      `yaml:"driver"`
	Internal bool        `yaml:"internal"`
	External interface{} `yaml:"external"`
	Name     string      `yaml:"name"`
}

type ComposePort struct {
	Raw       string
	Published string
	Target    uint32
}

type ComposeVolume struct {
	Source string
	Target string
	Type   string
	Raw    string
}

type ComposeStringSet []string

type ComposeServiceNetworks struct {
	Names      []string
	HasOptions bool
}

type ComposePresence bool

func (networks *ComposeServiceNetworks) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var stringsValue []string
	if err := unmarshal(&stringsValue); err == nil {
		networks.Names = stringsValue
		return nil
	}

	var mapValue map[string]any
	if err := unmarshal(&mapValue); err == nil {
		names := make([]string, 0, len(mapValue))
		hasOptions := false
		for key, value := range mapValue {
			names = append(names, key)
			if !isEmptyComposeValue(value) {
				hasOptions = true
			}
		}
		networks.Names = names
		networks.HasOptions = hasOptions
		return nil
	}

	var stringValue string
	if err := unmarshal(&stringValue); err == nil {
		networks.Names = []string{stringValue}
		return nil
	}

	return nil
}

func (networks ComposeServiceNetworks) Contains(name string) bool {
	return stringInSlice(name, networks.Names)
}

func (presence *ComposePresence) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}
	*presence = ComposePresence(!isEmptyComposeValue(raw))
	return nil
}

func (presence ComposePresence) Present() bool {
	return bool(presence)
}

func isEmptyComposeValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []interface{}:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	case map[interface{}]interface{}:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

var portRegex = regexp.MustCompile(`\$\{([^}]+)}`)

func (set *ComposeStringSet) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var stringsValue []string
	if err := unmarshal(&stringsValue); err == nil {
		*set = stringsValue
		return nil
	}

	var mapValue map[string]any
	if err := unmarshal(&mapValue); err == nil {
		values := make([]string, 0, len(mapValue))
		for key := range mapValue {
			values = append(values, key)
		}
		*set = values
		return nil
	}

	var stringValue string
	if err := unmarshal(&stringValue); err == nil {
		*set = []string{stringValue}
		return nil
	}

	*set = nil
	return nil
}

func (port *ComposePort) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var stringValue string
	if err := unmarshal(&stringValue); err == nil {
		parsed, err := parseComposePortString(stringValue)
		if err != nil {
			return err
		}
		*port = parsed
		return nil
	}

	var mapValue map[string]interface{}
	if err := unmarshal(&mapValue); err != nil {
		return err
	}

	published := fmt.Sprint(mapValue["published"])
	target, err := parseUint32Field(mapValue["target"])
	if err != nil {
		return fmt.Errorf("compose port target is invalid: %w", err)
	}
	if published == "<nil>" || published == "" {
		return fmt.Errorf("compose port target %d is missing published host port", target)
	}

	*port = ComposePort{
		Published: published,
		Target:    target,
	}
	return nil
}

func (volume *ComposeVolume) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var stringValue string
	if err := unmarshal(&stringValue); err == nil {
		parts := strings.Split(stringValue, ":")
		if len(parts) < 2 {
			return fmt.Errorf("compose volume %q must include source and target", stringValue)
		}
		volume.Raw = stringValue
		volume.Source = parts[0]
		volume.Target = parts[1]
		return nil
	}

	var mapValue map[string]interface{}
	if err := unmarshal(&mapValue); err != nil {
		return err
	}

	volume.Source = fmt.Sprint(mapValue["source"])
	if volume.Source == "<nil>" {
		volume.Source = fmt.Sprint(mapValue["src"])
	}
	if volume.Source == "<nil>" {
		volume.Source = ""
	}
	volume.Target = fmt.Sprint(mapValue["target"])
	if volume.Target == "<nil>" {
		volume.Target = fmt.Sprint(mapValue["dst"])
	}
	if volume.Target == "<nil>" {
		volume.Target = fmt.Sprint(mapValue["destination"])
	}
	if volume.Target == "<nil>" {
		volume.Target = ""
	}
	volume.Type = fmt.Sprint(mapValue["type"])
	if volume.Type == "<nil>" {
		volume.Type = ""
	}
	return nil
}

func parseComposePortString(raw string) (ComposePort, error) {
	port := ComposePort{Raw: raw}
	withoutProtocol := strings.SplitN(raw, "/", 2)[0]
	parts := strings.Split(withoutProtocol, ":")
	if len(parts) < 2 {
		return port, fmt.Errorf("port %s must publish a host port", raw)
	}

	port.Published = parts[len(parts)-2]
	target, err := strconv.ParseUint(parts[len(parts)-1], 10, 32)
	if err != nil {
		return port, fmt.Errorf("container port %q in %s is invalid", parts[len(parts)-1], raw)
	}
	port.Target = uint32(target)

	return port, nil
}

func parseUint32Field(value interface{}) (uint32, error) {
	switch v := value.(type) {
	case int:
		return uint32(v), nil
	case int64:
		return uint32(v), nil
	case uint64:
		return uint32(v), nil
	case string:
		parsed, err := strconv.ParseUint(v, 10, 32)
		return uint32(parsed), err
	default:
		return 0, fmt.Errorf("unexpected value %v", value)
	}
}

func loadCompose(composeFile string) (Compose, error) {
	data, err := os.ReadFile(composeFile)
	if err != nil {
		return Compose{}, fmt.Errorf("error while reading compose file: %w", err)
	}

	var raw Compose
	err = yaml.Unmarshal(data, &raw)
	if err != nil {
		return Compose{}, fmt.Errorf("error while parsing compose file: %s", err.Error())
	}
	if len(raw.Services) == 0 {
		return Compose{}, fmt.Errorf("compose file does not define any services")
	}
	return raw, nil
}

func ExtractPortsFromCompose(composeFile string) ([]string, error) {
	raw, err := loadCompose(composeFile)
	if err != nil {
		return nil, err
	}

	portVariables := make([]string, 0)
	seen := make(map[string]bool)
	for _, service := range raw.Services {
		for _, port := range service.Ports {
			varName, err := extractSinglePortVariable(port.Published)
			if err != nil {
				return nil, err
			}

			if !seen[varName] {
				seen[varName] = true
				portVariables = append(portVariables, varName)
			}
		}
	}

	return portVariables, nil
}

func ValidateInstancedComposeSSHContract(composeFile, defaultPortVar, hydraNetworkName, sshServiceName string, sshPort uint32, strictSadServers ...bool) error {
	raw, err := loadCompose(composeFile)
	if err != nil {
		return err
	}
	strict := len(strictSadServers) > 0 && strictSadServers[0]

	sshService, ok := raw.Services[sshServiceName]
	if !ok {
		return fmt.Errorf("instanced compose challenges must define service %q", sshServiceName)
	}

	exposedNetworkKey, err := findHydraNetworkKey(raw.Networks, hydraNetworkName)
	if err != nil {
		return err
	}

	internalNetworkKey, err := findSingleInternalNetworkKey(raw.Networks, exposedNetworkKey)
	if err != nil {
		return err
	}

	for serviceName, service := range raw.Services {
		if err := validateSafeComposeService(serviceName, service); err != nil {
			return err
		}
		if strict {
			if err := validateSadServersComposeService(serviceName, service); err != nil {
				return err
			}
		}

		if serviceName != sshServiceName && len(service.Ports) > 0 {
			return fmt.Errorf("service %q must not publish host ports; only %q may publish SSH", serviceName, sshServiceName)
		}

		if strict {
			if err := validateSadServersServiceNetworks(serviceName, service.Networks, exposedNetworkKey, internalNetworkKey, sshServiceName); err != nil {
				return err
			}
			continue
		}

		hasHydra := service.Networks.Contains(exposedNetworkKey)
		hasInternal := service.Networks.Contains(internalNetworkKey)
		if serviceName == sshServiceName {
			if !hasHydra {
				return fmt.Errorf("service %q must attach to external network %q", sshServiceName, exposedNetworkKey)
			}
			if !hasInternal {
				return fmt.Errorf("service %q must attach to internal network %q", sshServiceName, internalNetworkKey)
			}
		} else {
			if hasHydra {
				return fmt.Errorf("service %q must not attach to external network %q", serviceName, exposedNetworkKey)
			}
			if !hasInternal {
				return fmt.Errorf("service %q must attach to internal network %q", serviceName, internalNetworkKey)
			}
		}
	}

	if len(sshService.Ports) != 1 {
		return fmt.Errorf("service %q must publish exactly one host port for SSH", sshServiceName)
	}

	sshPublishedVar, err := extractSinglePortVariable(sshService.Ports[0].Published)
	if err != nil {
		return err
	}
	if sshService.Ports[0].Target != sshPort {
		return fmt.Errorf("service %q must map %s to container port %d", sshServiceName, defaultPortVar, sshPort)
	}
	if sshPublishedVar != defaultPortVar {
		return fmt.Errorf("default_port_var %q must match SSH published port variable %q", defaultPortVar, sshPublishedVar)
	}

	if !serviceHasChallengeNamedVolume(sshService, raw.Volumes) {
		return fmt.Errorf("service %q must mount a named compose volume at %s", sshServiceName, ComposeChallengeMountTarget)
	}

	if strict {
		if err := validateSadServersNetworks(raw.Networks, exposedNetworkKey, internalNetworkKey); err != nil {
			return err
		}
		if err := validateSadServersCheckerFiles(composeFile); err != nil {
			return err
		}
	}

	return nil
}

func extractSinglePortVariable(port string) (string, error) {
	matches := portRegex.FindAllStringSubmatch(port, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("port %s is not mapped using an env variable", port)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("port %s must contain exactly one env variable", port)
	}

	varName := matches[0][1]
	if strings.Contains(varName, ":-") {
		return "", fmt.Errorf("port variable ${%s} uses default value syntax (:-) which is not supported; use ${%s} instead",
			varName, strings.SplitN(varName, ":-", 2)[0])
	}
	return varName, nil
}

func findHydraNetworkKey(networks map[string]ComposeNetwork, hydraNetworkName string) (string, error) {
	for key, network := range networks {
		if !network.ExternalEnabled() {
			continue
		}
		name := network.Name
		if name == "" {
			name = key
		}
		if name == hydraNetworkName {
			return key, nil
		}
	}

	return "", fmt.Errorf("compose file must define an external network named %q", hydraNetworkName)
}

func findSingleInternalNetworkKey(networks map[string]ComposeNetwork, exposedNetworkKey string) (string, error) {
	internalNetworks := make([]string, 0)
	for key, network := range networks {
		if key == exposedNetworkKey {
			continue
		}
		if network.Internal {
			internalNetworks = append(internalNetworks, key)
		}
	}
	if len(internalNetworks) != 1 {
		return "", fmt.Errorf("compose file must define exactly one non-exposed internal network")
	}
	return internalNetworks[0], nil
}

func (network ComposeNetwork) ExternalEnabled() bool {
	switch v := network.External.(type) {
	case bool:
		return v
	case map[interface{}]interface{}:
		enabled, ok := v["external"].(bool)
		return ok && enabled
	case map[string]interface{}:
		enabled, ok := v["external"].(bool)
		return ok && enabled
	default:
		return false
	}
}

func validateSafeComposeService(serviceName string, service ComposeService) error {
	if service.NetworkMode == "host" {
		return fmt.Errorf("service %q must not use network_mode: host", serviceName)
	}
	if service.Privileged {
		return fmt.Errorf("service %q must not use privileged: true", serviceName)
	}
	for _, cap := range service.CapAdd {
		normalized := strings.ToUpper(strings.TrimSpace(cap))
		if normalized == "NET_ADMIN" || normalized == "CAP_NET_ADMIN" {
			return fmt.Errorf("service %q must not add NET_ADMIN capability", serviceName)
		}
	}
	return nil
}

func validateSadServersComposeService(serviceName string, service ComposeService) error {
	if strings.TrimSpace(service.NetworkMode) != "" {
		return fmt.Errorf("service %q must not set network_mode", serviceName)
	}
	if service.PIDMode == "host" {
		return fmt.Errorf("service %q must not use pid: host", serviceName)
	}
	if service.IPCMode == "host" {
		return fmt.Errorf("service %q must not use ipc: host", serviceName)
	}
	if service.UTSMode == "host" {
		return fmt.Errorf("service %q must not use uts: host", serviceName)
	}
	if strings.TrimSpace(service.UsernsMode) != "" {
		return fmt.Errorf("service %q must not set userns_mode", serviceName)
	}
	if strings.TrimSpace(service.CgroupnsMode) != "" {
		return fmt.Errorf("service %q must not set cgroupns_mode", serviceName)
	}
	if strings.TrimSpace(service.CgroupParent) != "" {
		return fmt.Errorf("service %q must not set cgroup_parent", serviceName)
	}
	if len(service.Devices) > 0 {
		return fmt.Errorf("service %q must not declare devices", serviceName)
	}
	if service.ExtraHosts.Present() {
		return fmt.Errorf("service %q must not set extra_hosts", serviceName)
	}
	if service.VolumesFrom.Present() {
		return fmt.Errorf("service %q must not set volumes_from", serviceName)
	}
	if service.EnvFile.Present() {
		return fmt.Errorf("service %q must not set env_file", serviceName)
	}
	if service.Secrets.Present() {
		return fmt.Errorf("service %q must not set secrets", serviceName)
	}
	if service.Configs.Present() {
		return fmt.Errorf("service %q must not set configs", serviceName)
	}
	if service.Sysctls.Present() {
		return fmt.Errorf("service %q must not set sysctls", serviceName)
	}
	if service.Ulimits.Present() {
		return fmt.Errorf("service %q must not set ulimits", serviceName)
	}
	if len(service.CapAdd) > 0 {
		return fmt.Errorf("service %q must not add Linux capabilities", serviceName)
	}
	for _, opt := range service.SecurityOpt {
		normalized := strings.ToLower(strings.TrimSpace(opt))
		if normalized != "" && normalized != "no-new-privileges:true" {
			return fmt.Errorf("service %q must not set security_opt %q", serviceName, opt)
		}
	}
	for _, volume := range service.Volumes {
		if volume.Target == "/var/run/docker.sock" || volume.Source == "/var/run/docker.sock" {
			return fmt.Errorf("service %q must not mount Docker socket", serviceName)
		}
		if isHostBindMount(volume) {
			return fmt.Errorf("service %q must not use host bind mount %q", serviceName, volume.Raw)
		}
	}
	return nil
}

func validateSadServersNetworks(networks map[string]ComposeNetwork, exposedNetworkKey, internalNetworkKey string) error {
	for key := range networks {
		if key == exposedNetworkKey || key == internalNetworkKey {
			continue
		}
		return fmt.Errorf("sadservers compose must not declare extra network %q", key)
	}
	if len(networks) != 2 {
		return fmt.Errorf("sadservers compose must define exactly the hydra network and one internal network")
	}
	return nil
}

func validateSadServersServiceNetworks(serviceName string, networks ComposeServiceNetworks, exposedNetworkKey, internalNetworkKey, sshServiceName string) error {
	if networks.HasOptions {
		return fmt.Errorf("service %q must not set per-service network options", serviceName)
	}

	if serviceName == sshServiceName {
		if !sameStringSet(networks.Names, []string{exposedNetworkKey, internalNetworkKey}) {
			return fmt.Errorf("service %q must attach exactly to networks %q and %q", serviceName, exposedNetworkKey, internalNetworkKey)
		}
		return nil
	}

	if !sameStringSet(networks.Names, []string{internalNetworkKey}) {
		return fmt.Errorf("service %q must attach exactly to internal network %q", serviceName, internalNetworkKey)
	}
	return nil
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]bool{}
	for _, value := range got {
		seen[value] = true
	}
	for _, value := range want {
		if !seen[value] {
			return false
		}
	}
	return true
}

func validateSadServersCheckerFiles(composeFile string) error {
	checkPath := filepath.Join(filepath.Dir(composeFile), "check.sh")
	info, err := os.Stat(checkPath)
	if err != nil {
		return fmt.Errorf("sadservers challenges must provide root-level check.sh: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("sadservers check.sh must be a regular file")
	}
	return nil
}

func isHostBindMount(volume ComposeVolume) bool {
	source := strings.TrimSpace(volume.Source)
	if source == "" {
		return false
	}
	if volume.Type == "bind" {
		return true
	}
	if strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") {
		return true
	}
	return strings.Contains(source, string(os.PathSeparator))
}

func serviceHasChallengeNamedVolume(service ComposeService, volumes map[string]any) bool {
	for _, volume := range service.Volumes {
		if volume.Target != ComposeChallengeMountTarget {
			continue
		}
		if volume.Source == "" || strings.HasPrefix(volume.Source, ".") || strings.HasPrefix(volume.Source, "/") {
			return false
		}
		if volume.Type != "" && volume.Type != "volume" {
			return false
		}
		if volumes == nil {
			return true
		}
		_, ok := volumes[volume.Source]
		return ok
	}
	return false
}

func stringInSlice(value string, items []string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
