package cache

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/utils"
	"strconv"
)

const reservePortsScript = `
local hostKey = KEYS[1]
local firstPort = tonumber(ARGV[1])
local portRange = tonumber(ARGV[2])
local count = tonumber(ARGV[3])
local selected = {}

for offset = 0, portRange - 1 do
	local port = firstPort + offset
	if redis.call("SISMEMBER", hostKey, port) == 0 then
		table.insert(selected, port)
		if #selected == count then
			break
		end
	end
end

if #selected < count then
	return {}
end

for _, port in ipairs(selected) do
	redis.call("SADD", hostKey, port)
end

return selected
`

// GetFreePortOnHost gets the first available port in the specific range by checking its existance in the cache.
// algorithm can be imprived later on if it bottlenecks performance.
func GetFreePortOnHost(host string, firstPort uint32, portRange uint32) (uint32, error) {
	ports, err := GetFreePortsOnHost(host, firstPort, portRange, 1)
	if err != nil {
		return 0, err
	}
	if len(ports) == 0 {
		return 0, fmt.Errorf("no free port found on host: %s", host)
	}

	return ports[0], nil
}

func GetFreePortsOnHost(host string, firstPort uint32, portRange uint32, count int) ([]uint32, error) {
	if Cache == nil {
		Init()
	}

	if count <= 0 {
		return []uint32{}, nil
	}

	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	ctx := context.Background()
	hostKey := utils.HostToKey(host)

	result, err := Cache.Eval(ctx, reservePortsScript, []string{hostKey}, firstPort, portRange, count).Result()
	if err != nil {
		return nil, err
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != count {
		return nil, fmt.Errorf("no free port found on host: %s", host)
	}

	ports := make([]uint32, len(values))
	for i, value := range values {
		port, err := redisValueToUint32(value)
		if err != nil {
			return nil, err
		}
		ports[i] = port
	}

	return ports, nil
}

// AssignFreePortOnHostToContainer allocates a port for a container on a given host machine
func AssignFreePortOnHostToContainer(host string, containerId string, port uint32) error {
	return AssignPortsOnHostToContainer(host, containerId, []uint32{port})
}

func AssignPortsOnHostToContainer(host string, containerId string, ports []uint32) error {
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	ctx := context.Background()
	instanceKey := utils.ContainerToKey(host, containerId)

	pipe := Cache.TxPipeline()
	for _, port := range ports {
		pipe.SAdd(ctx, instanceKey, port)
	}

	results, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	for i, result := range results {
		cmd, ok := result.(*redis.IntCmd)
		if !ok {
			continue
		}
		added, err := cmd.Result()
		if err != nil {
			return err
		}
		if added == 0 {
			return fmt.Errorf("port: %v on host: %s is already registered to instance: %s", ports[i], host, containerId)
		}
	}

	return nil
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

func redisValueToUint32(value interface{}) (uint32, error) {
	switch v := value.(type) {
	case int64:
		return uint32(v), nil
	case string:
		port, err := strconv.ParseUint(v, 10, 32)
		return uint32(port), err
	case []byte:
		port, err := strconv.ParseUint(string(v), 10, 32)
		return uint32(port), err
	default:
		return 0, fmt.Errorf("unexpected Redis port value %T", value)
	}
}

func FreePortOnHost(host string, port uint32) error {
	if Cache == nil {
		Init()
	}

	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	ctx := context.Background()
	hostKey := utils.HostToKey(host)

	_, err := Cache.SRem(ctx, hostKey, port).Result()
	if err != nil {
		return err
	}

	return nil
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
