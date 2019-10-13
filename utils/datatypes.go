package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const mappingDelimeter = ":"

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

// ParsePortMapping parses the port mapping string and return the required ports
// If the portMapping string is not valid, this returns an error.
// The format of the port mapping is `HOST_PORT:CONTAINER_PORT`
func ParsePortMapping(portMap string) (uint32, uint32, error) {
	ports := strings.Split(portMap, mappingDelimeter)

	if len(ports) != 2 {
		return 0, 0, errors.New("port mapping string is not valid")
	}

	hostPort, err := strconv.ParseUint(ports[0], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("host port is not a valid port in: %s", portMap)
	}

	containerPort, err := strconv.ParseUint(ports[1], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("container port is not a valid port in: %s", portMap)
	}

	return uint32(hostPort), uint32(containerPort), nil
}
