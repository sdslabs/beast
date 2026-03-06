package cache

import (
	"context"
	"fmt"
	"github.com/sdslabs/beastv4/utils"
	"strconv"
)

// GetFreePortOnHost gets the first available port in the specific range by checking its existance in the cache.
// algorithm can be imprived later on if it bottlenecks performance.
func GetFreePortOnHost(host string, firstPort uint32, portRange uint32) (uint32, error) {
	if Cache == nil {
		Init()
	}

	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	ctx := context.Background()
	hostKey := utils.HostToKey(host)

	for i := range portRange {
		port := firstPort + i
		result, err := Cache.SAdd(ctx, hostKey, port).Result()
		if err != nil {
			return 0, err
		}

		if result == 1 {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no free port found on host: %s", host)
}

// AssignFreePortOnHostToContainer allocates a port for a container on a given host machine
func AssignFreePortOnHostToContainer(host string, containerId string, port uint32) error {
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	ctx := context.Background()
	instanceKey := utils.ContainerToKey(host, containerId)

	result, err := Cache.SAdd(ctx, instanceKey, port).Result()
	if err != nil {
		return err
	}

	if result == 1 {
		return nil
	}

	return fmt.Errorf("port: %v on host: %s is already registered to instance: %s", port, host, containerId)
}

// GetContainerPortsOnHost gets all the assigned ports for a given container on a given host
func GetContainerPortsOnHost(host string, containerId string) ([]uint32, error) {
	if Cache == nil {
		Init()
	}

	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	ctx := context.Background()
	instanceKey := utils.ContainerToKey(host, containerId)

	result, err := Cache.SMembers(ctx, instanceKey).Result()
	if err != nil {
		return nil, err
	}

	ports := make([]uint32, len(result))
	for i, s := range result {
		port, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return nil, err
		}

		ports[i] = uint32(port)
	}

	return ports, nil
}

// FreeContainerPortsOnHost frees all allocated host ports on a machine, at present occupied by a container
func FreeContainerPortsOnHost(host string, containerId string) error {
	if Cache == nil {
		Init()
	}

	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	ctx := context.Background()
	hostKey := utils.HostToKey(host)
	instanceKey := utils.ContainerToKey(host, containerId)

	result, err := Cache.SMembers(ctx, instanceKey).Result()
	if err != nil {
		return err
	}

	ports := make([]uint32, len(result))
	for i, portString := range result {
		port, err := strconv.ParseUint(portString, 10, 32)
		if err != nil {
			return err
		}

		ports[i] = uint32(port)
		Cache.SRem(ctx, instanceKey, port)
	}

	for _, port := range ports {
		_, err = Cache.SRem(ctx, hostKey, port).Result()
		if err != nil {
			return err
		}
	}

	return nil
}
