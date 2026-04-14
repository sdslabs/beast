package utils

import (
	"errors"
	"fmt"
	"github.com/sdslabs/beastv4/core"
	"strconv"
	"strings"
)

// From a list of strings generate a list containing only unique strings
// from the list.
func GetUniqueStrings(list []string) []string {
	var uniq []string
	m := make(map[string]bool)

	for _, str := range list {
		if _, ok := m[str]; !ok {
			m[str] = true
			uniq = append(uniq, str)
		}
	}

	return uniq
}

// Returns true if in slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func UInt32InList(a uint32, list []uint32) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func Uint32InIndexList(a uint32, la []uint32, lb []uint32) (bool, uint32) {
	for i, a_ := range la {
		if a == a_ {
			return true, lb[i]
		}
	}

	return false, 0
}

// ParsePortMapping parses the port mapping string and return the required ports
// If the portMapping string is not valid, this returns an error.
// The format of the port mapping is `PORT_FIRST:PORT_LAST`
func ParsePortMapping(portMap string) (uint32, uint32, error) {
	ports := strings.Split(portMap, core.MappingDelimiter)

	if len(ports) != 2 {
		return 0, 0, errors.New("port mapping string is not valid")
	}

	firstPort, err := strconv.ParseUint(ports[0], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("first port is not a valid port in: %s", portMap)
	}

	lastPort, err := strconv.ParseUint(ports[1], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("second port is not a valid port in: %s", portMap)
	}

	if firstPort > lastPort {
		return 0, 0, fmt.Errorf("first port is greater than last port")
	}

	return uint32(firstPort), uint32(lastPort), nil
}

func PortMappingToEnvironmentVariable(ports map[string]uint32) string {
	env := make([]string, len(ports))

	i := 0
	for variable, port := range ports {
		env[i] = fmt.Sprintf("%s=%s", variable, strconv.FormatUint(uint64(port), 10))
		i++
	}

	return strings.Join(env, " ")
}
