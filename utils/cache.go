package utils

import "fmt"

func HostToKey(host string) string {
	return fmt.Sprintf("host:%s", host)
}

func ContainerToKey(host string, containerId string) string {
	return fmt.Sprintf("host:%s:container:%s", host, containerId)
}
