package cache

import (
	"context"
	"fmt"
	"github.com/sdslabs/beastv4/utils"
	"strconv"
)

func GetFreePort(host string, firstPort uint32, portRange uint32) (uint32, error) {
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

func RegisterFreePort(host string, containerId string, port uint32) error {
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

func GetContainerPorts(host string, containerId string) ([]uint32, error) {
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

func FreeContainerPorts(host string, containerId string) error {
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
