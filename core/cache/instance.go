package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type Instance struct {
	InstanceID     string    `json:"instance_id"`
	ChallengeName  string    `json:"challenge_name"`
	ContainerID    string    `json:"container_id"`
	HostedAddress  string    `json:"hosted_address"`
	CheckHash      string    `json:"check_hash"`
	Port           uint32    `json:"port"`
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	DeploymentType string    `json:"deployment_type"`
	ServerDeployed string    `json:"server_deployed"`
}

const (
	InstanceKeyPrefix     = "beast:instance:"
	UserInstanceKeyPrefix = "beast:user_instance:"
	InstancesSetKey       = "beast:instances"
	InstanceDeletionQueue = "beast:instances:to_delete"
)

func instanceKey(instanceID string) string {
	return InstanceKeyPrefix + instanceID
}

func userInstanceKey(userID, challengeName string) string {
	return UserInstanceKeyPrefix + userID + ":" + challengeName
}

func SaveInstance(instance *Instance, ttl time.Duration) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	data, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal instance: %w", err)
	}

	key := instanceKey(instance.InstanceID)
	err = Cache.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to save instance: %w", err)
	}

	userKey := userInstanceKey(instance.UserID, instance.ChallengeName)
	err = Cache.Set(ctx, userKey, instance.InstanceID, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to save user instance mapping: %w", err)
	}

	err = Cache.SAdd(ctx, InstancesSetKey, instance.InstanceID).Err()
	if err != nil {
		log.Warnf("failed to add instance to set: %v", err)
	}

	log.Debugf("Saved instance %s for user %s, challenge %s, port %d, expires in %v",
		instance.InstanceID, instance.UserID, instance.ChallengeName, instance.Port, ttl)

	return nil
}

func GetInstance(instanceID string) (*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	key := instanceKey(instanceID)
	data, err := Cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	var instance Instance
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	return &instance, nil
}

func GetUserInstance(userID, challengeName string) (*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	userKey := userInstanceKey(userID, challengeName)
	instanceID, err := Cache.Get(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("user instance not found: %w", err)
	}

	key := instanceKey(instanceID)
	data, err := Cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	var instance Instance
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	return &instance, nil
}

func GetUserInstances(userID string) ([]*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	pattern := UserInstanceKeyPrefix + userID + ":*"
	var instances []*Instance

	iter := Cache.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		userKey := iter.Val()
		instanceID, err := Cache.Get(ctx, userKey).Result()
		if err != nil {
			continue
		}

		key := instanceKey(instanceID)
		data, err := Cache.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var instance Instance
		err = json.Unmarshal(data, &instance)
		if err != nil {
			continue
		}

		instances = append(instances, &instance)
	}

	return instances, nil
}

func GetAllInstances() ([]*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	instanceIDs, err := Cache.SMembers(ctx, InstancesSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance IDs: %w", err)
	}

	var instances []*Instance
	for _, id := range instanceIDs {
		key := instanceKey(id)
		data, err := Cache.Get(ctx, key).Bytes()
		if err != nil {
			Cache.SRem(ctx, InstancesSetKey, id)
			continue
		}

		var instance Instance
		err = json.Unmarshal(data, &instance)
		if err != nil {
			continue
		}

		instances = append(instances, &instance)
	}

	return instances, nil
}

// GetChallengeInstances retrieves all active instances for a specific challenge
func GetChallengeInstances(challengeName string) ([]*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	instanceIDs, err := Cache.SMembers(ctx, InstancesSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance IDs: %w", err)
	}

	var instances []*Instance
	for _, id := range instanceIDs {
		key := instanceKey(id)
		data, err := Cache.Get(ctx, key).Bytes()
		if err != nil {
			Cache.SRem(ctx, InstancesSetKey, id)
			continue
		}

		var instance Instance
		err = json.Unmarshal(data, &instance)
		if err != nil {
			continue
		}

		if instance.ChallengeName == challengeName {
			instances = append(instances, &instance)
		}
	}

	return instances, nil
}

func DeleteInstance(instanceID string) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	key := instanceKey(instanceID)
	data, err := Cache.Get(ctx, key).Bytes()
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	var instance Instance
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	err = Cache.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	userKey := userInstanceKey(instance.UserID, instance.ChallengeName)
	Cache.Del(ctx, userKey)
	Cache.SRem(ctx, InstancesSetKey, instanceID)

	log.Debugf("Deleted instance %s for user %s, challenge %s",
		instanceID, instance.UserID, instance.ChallengeName)

	return nil
}

func ExtendInstance(instanceID string, additionalTime time.Duration) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	key := instanceKey(instanceID)

	data, err := Cache.Get(ctx, key).Bytes()
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	var instance Instance
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	newExpiresAt := instance.ExpiresAt.Add(additionalTime)
	instance.ExpiresAt = newExpiresAt

	newTTL := time.Until(newExpiresAt)
	if newTTL <= 0 {
		return fmt.Errorf("instance has already expired")
	}

	updatedData, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal instance: %w", err)
	}

	err = Cache.Set(ctx, key, updatedData, newTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to extend instance: %w", err)
	}

	userKey := userInstanceKey(instance.UserID, instance.ChallengeName)
	Cache.Expire(ctx, userKey, newTTL)

	log.Debugf("Extended instance %s by %v, new expiration: %v", instanceID, additionalTime, newExpiresAt)

	return nil
}

func CountUserInstances(userID string) (int, error) {
	if Cache == nil {
		return 0, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	pattern := UserInstanceKeyPrefix + userID + ":*"
	count := 0

	iter := Cache.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		count++
	}

	return count, nil
}

func GetInstanceTTL(instanceID string) (time.Duration, error) {
	if Cache == nil {
		return 0, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	key := instanceKey(instanceID)
	ttl, err := Cache.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

func QueueInstanceForDeletion(instanceID string) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	key := instanceKey(instanceID)

	data, err := Cache.Get(ctx, key).Bytes()
	if err != nil {
		Cache.SRem(ctx, InstancesSetKey, instanceID)
		return fmt.Errorf("instance not found: %w", err)
	}

	var instance Instance
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	pipe := Cache.TxPipeline()
	pipe.LPush(ctx, InstanceDeletionQueue, data)
	pipe.SRem(ctx, InstancesSetKey, instanceID)

	userKey := userInstanceKey(instance.UserID, instance.ChallengeName)
	pipe.Del(ctx, userKey)
	pipe.Del(ctx, key)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to queue instance for deletion: %w", err)
	}

	log.Debugf("Queued instance %s for deletion (user: %s, challenge: %s)",
		instanceID, instance.UserID, instance.ChallengeName)

	return nil
}

func PopInstanceForDeletion() (*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	data, err := Cache.RPop(ctx, InstanceDeletionQueue).Bytes()
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to pop from deletion queue: %w", err)
	}

	var instance Instance
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance from queue: %w", err)
	}

	log.Debugf("Popped instance %s from deletion queue", instance.InstanceID)
	return &instance, nil
}

func GetDeletionQueueLength() (int64, error) {
	if Cache == nil {
		return 0, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	return Cache.LLen(ctx, InstanceDeletionQueue).Result()
}

func GetExpiredInstances() ([]*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	instanceIDs, err := Cache.SMembers(ctx, InstancesSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance IDs: %w", err)
	}

	now := time.Now()
	var expired []*Instance

	for _, id := range instanceIDs {
		key := instanceKey(id)
		data, err := Cache.Get(ctx, key).Bytes()
		if err != nil {
			Cache.SRem(ctx, InstancesSetKey, id)
			continue
		}

		var instance Instance
		err = json.Unmarshal(data, &instance)
		if err != nil {
			continue
		}

		if instance.ExpiresAt.Before(now) {
			expired = append(expired, &instance)
		}
	}

	return expired, nil
}
