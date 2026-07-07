package utils

import (
	"fmt"
	"strings"
)

const (
	instanceKeyPrefix     = "beast:instance"
	instanceExpiryPrefix  = "beast:instance_expiry"
	userInstanceKeyPrefix = "beast:user_instance"
	InstancesSetKey       = "beast:instances"
	InstanceDeletionQueue = "beast:instances:to_delete"

	hostPrefixKey      = "beast:host"
	containerPrefixKey = "container"
)

func HostToKey(host string) string {
	return fmt.Sprintf("%s:%s", hostPrefixKey, host)
}

func ContainerToKey(host string, containerId string) string {
	return fmt.Sprintf("%s:%s:%s:%s", hostPrefixKey, host, containerPrefixKey, containerId)
}

func InstanceToKey(instanceID string) string {
	return fmt.Sprintf("%s:%s", instanceKeyPrefix, instanceID)
}

func InstanceExpiryToKey(instanceID string) string {
	return fmt.Sprintf("%s:%s", instanceExpiryPrefix, instanceID)
}

func InstanceIDFromExpiryKey(key string) (string, bool) {
	prefix := instanceExpiryPrefix + ":"
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}

	instanceID := strings.TrimPrefix(key, prefix)
	return instanceID, instanceID != ""
}

func UserChallengeToKey(userID, challengeName string) string {
	return fmt.Sprintf("%s:%s:%s", userInstanceKeyPrefix, userID, challengeName)
}

func UserChallengesAllKey(userID string) string {
	return fmt.Sprintf("%s:%s:*", userInstanceKeyPrefix, userID)
}
