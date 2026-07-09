package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sdslabs/beastv4/utils"
	"time"

	log "github.com/sirupsen/logrus"
)

type Instance struct {
	InstanceID      string    `json:"instance_id"`
	ChallengeName   string    `json:"challenge_name"`
	ContainerID     string    `json:"container_id"`
	PortOwner       string    `json:"port_owner"`
	CheckHash       string    `json:"check_hash"`
	CheckManifest   string    `json:"check_manifest"`
	CheckerImageID  string    `json:"checker_image_id"`
	CheckerImageRef string    `json:"checker_image_ref"`
	CheckerMode     string    `json:"checker_mode"`
	Port            uint32    `json:"port"`
	UserID          string    `json:"user_id"`
	Username        string    `json:"username"`
	State           string    `json:"state,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	DeploymentType  string    `json:"deployment_type"`
	ServerDeployed  string    `json:"server_deployed"`
}

const (
	InstanceStateActive   = "active"
	InstanceStateDeleting = "deleting"
)

type InstanceReservationStatus int

const (
	InstanceReservationGranted InstanceReservationStatus = iota
	InstanceReservationChallengeActive
	InstanceReservationLimitReached
)

type InstanceReservationResult struct {
	Status             InstanceReservationStatus
	ExistingInstanceID string
}

const reserveInstanceSlotScript = `
local user_key = KEYS[1]
local active_set_key = KEYS[2]
local reservation_key = KEYS[3]

local instance_id = ARGV[1]
local ttl_seconds = tonumber(ARGV[2])
local max_instances = tonumber(ARGV[3])
local instance_key_prefix = ARGV[4]
local reservation_key_prefix = ARGV[5]

local active_ids = redis.call("SMEMBERS", active_set_key)
for _, active_id in ipairs(active_ids) do
	local instance_exists = redis.call("EXISTS", instance_key_prefix .. active_id)
	local reservation_exists = redis.call("EXISTS", reservation_key_prefix .. active_id)
	if instance_exists == 0 and reservation_exists == 0 then
		redis.call("SREM", active_set_key, active_id)
	end
end

local existing_id = redis.call("GET", user_key)
if existing_id then
	local instance_exists = redis.call("EXISTS", instance_key_prefix .. existing_id)
	local reservation_exists = redis.call("EXISTS", reservation_key_prefix .. existing_id)
	if instance_exists == 1 or reservation_exists == 1 then
		return {1, existing_id}
	end

	redis.call("DEL", user_key)
	redis.call("SREM", active_set_key, existing_id)
end

if max_instances > 0 and redis.call("SCARD", active_set_key) >= max_instances then
	return {2, ""}
end

if not redis.call("SET", user_key, instance_id, "EX", ttl_seconds, "NX") then
	local current_id = redis.call("GET", user_key)
	if current_id then
		return {1, current_id}
	end
	return {1, ""}
end

redis.call("SET", reservation_key, "1", "EX", ttl_seconds)
redis.call("SADD", active_set_key, instance_id)
return {0, instance_id}
`

const releaseInstanceReservationScript = `
local user_key = KEYS[1]
local active_set_key = KEYS[2]
local reservation_key = KEYS[3]
local instance_id = ARGV[1]

if redis.call("GET", user_key) == instance_id then
	redis.call("DEL", user_key)
end

redis.call("DEL", reservation_key)
redis.call("SREM", active_set_key, instance_id)
return 1
`

const queueInstanceForDeletionScript = `
local instance_key = KEYS[1]
local queue_key = KEYS[2]
local deletion_set_key = KEYS[3]
local instances_set_key = KEYS[4]
local user_key = KEYS[5]
local expiry_key = KEYS[6]
local active_set_key = KEYS[7]
local reservation_key = KEYS[8]

local instance_id = ARGV[1]
local instance_data = ARGV[2]

redis.call("SET", instance_key, instance_data)
redis.call("SADD", instances_set_key, instance_id)
redis.call("SADD", active_set_key, instance_id)
redis.call("SET", user_key, instance_id)
redis.call("DEL", expiry_key)
redis.call("DEL", reservation_key)

local added = redis.call("SADD", deletion_set_key, instance_id)
if added == 1 then
	redis.call("LPUSH", queue_key, instance_data)
end

return added
`

func (instance *Instance) PortOwnerID() string {
	if instance.PortOwner != "" {
		return instance.PortOwner
	}

	return instance.ContainerID
}

func (instance *Instance) StateOrActive() string {
	if instance == nil || instance.State == "" {
		return InstanceStateActive
	}
	return instance.State
}

func (instance *Instance) ActiveAt(now time.Time) bool {
	return instance != nil && instance.StateOrActive() == InstanceStateActive && instance.ExpiresAt.After(now)
}

func ReserveInstanceSlot(userID, challengeName, instanceID string, ttl time.Duration, maxInstancesPerUser int) (InstanceReservationResult, error) {
	if Cache == nil {
		return InstanceReservationResult{}, fmt.Errorf("redis cache not initialized")
	}
	if ttl <= 0 {
		return InstanceReservationResult{}, fmt.Errorf("instance ttl must be positive")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	ttlSeconds := int64(ttl / time.Second)
	if ttlSeconds <= 0 {
		ttlSeconds = 1
	}

	userKey := utils.UserChallengeToKey(userID, challengeName)
	activeSetKey := utils.UserActiveInstancesToKey(userID)
	reservationKey := utils.InstanceReservationToKey(instanceID)

	raw, err := Cache.Eval(ctx, reserveInstanceSlotScript, []string{
		userKey,
		activeSetKey,
		reservationKey,
	}, instanceID, ttlSeconds, maxInstancesPerUser, utils.InstanceKeyPrefix(), utils.InstanceReservationKeyPrefix()).Result()
	if err != nil {
		return InstanceReservationResult{}, fmt.Errorf("failed to reserve instance slot: %w", err)
	}

	values, ok := raw.([]interface{})
	if !ok || len(values) < 2 {
		return InstanceReservationResult{}, fmt.Errorf("unexpected instance reservation response: %#v", raw)
	}

	status, err := redisInt(values[0])
	if err != nil {
		return InstanceReservationResult{}, fmt.Errorf("unexpected instance reservation status: %w", err)
	}

	return InstanceReservationResult{
		Status:             InstanceReservationStatus(status),
		ExistingInstanceID: fmt.Sprint(values[1]),
	}, nil
}

func ReleaseInstanceReservation(userID, challengeName, instanceID string) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	userKey := utils.UserChallengeToKey(userID, challengeName)
	activeSetKey := utils.UserActiveInstancesToKey(userID)
	reservationKey := utils.InstanceReservationToKey(instanceID)

	return Cache.Eval(ctx, releaseInstanceReservationScript, []string{
		userKey,
		activeSetKey,
		reservationKey,
	}, instanceID).Err()
}

func redisInt(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case uint64:
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("%T", value)
	}
}

func SaveInstance(instance *Instance, ttl time.Duration) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}
	if instance.State == "" {
		instance.State = InstanceStateActive
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	data, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal instance: %w", err)
	}

	key := utils.InstanceToKey(instance.InstanceID)
	expiryKey := utils.InstanceExpiryToKey(instance.InstanceID)
	userKey := utils.UserChallengeToKey(instance.UserID, instance.ChallengeName)
	activeSetKey := utils.UserActiveInstancesToKey(instance.UserID)
	reservationKey := utils.InstanceReservationToKey(instance.InstanceID)

	pipe := Cache.TxPipeline()
	pipe.Set(ctx, key, data, 0)
	pipe.Set(ctx, expiryKey, instance.InstanceID, ttl)
	pipe.Set(ctx, userKey, instance.InstanceID, ttl)
	pipe.SAdd(ctx, utils.InstancesSetKey, instance.InstanceID)
	pipe.SAdd(ctx, activeSetKey, instance.InstanceID)
	pipe.Del(ctx, reservationKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save instance: %w", err)
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

	key := utils.InstanceToKey(instanceID)
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

	userKey := utils.UserChallengeToKey(userID, challengeName)
	instanceID, err := Cache.Get(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("user instance not found: %w", err)
	}

	key := utils.InstanceToKey(instanceID)
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

	var instances []*Instance

	instanceIDs, err := pruneUserActiveInstancesLocked(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, instanceID := range instanceIDs {
		key := utils.InstanceToKey(instanceID)
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

	instanceIDs, err := Cache.SMembers(ctx, utils.InstancesSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance IDs: %w", err)
	}

	var instances []*Instance
	for _, id := range instanceIDs {
		key := utils.InstanceToKey(id)
		data, err := Cache.Get(ctx, key).Bytes()
		if err != nil {
			Cache.SRem(ctx, utils.InstancesSetKey, id)
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

func GetChallengeInstances(challengeName string) ([]*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	instanceIDs, err := Cache.SMembers(ctx, utils.InstancesSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance IDs: %w", err)
	}

	var instances []*Instance
	for _, id := range instanceIDs {
		key := utils.InstanceToKey(id)
		data, err := Cache.Get(ctx, key).Bytes()
		if err != nil {
			Cache.SRem(ctx, utils.InstancesSetKey, id)
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

	key := utils.InstanceToKey(instanceID)
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

	userKey := utils.UserChallengeToKey(instance.UserID, instance.ChallengeName)
	expiryKey := utils.InstanceExpiryToKey(instanceID)
	activeSetKey := utils.UserActiveInstancesToKey(instance.UserID)
	reservationKey := utils.InstanceReservationToKey(instanceID)
	Cache.Del(ctx, userKey)
	Cache.Del(ctx, expiryKey)
	Cache.Del(ctx, reservationKey)
	Cache.SRem(ctx, utils.InstancesSetKey, instanceID)
	Cache.SRem(ctx, utils.InstanceDeletionSet, instanceID)
	Cache.SRem(ctx, activeSetKey, instanceID)

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

	key := utils.InstanceToKey(instanceID)

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

	err = Cache.Set(ctx, key, updatedData, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to extend instance: %w", err)
	}

	userKey := utils.UserChallengeToKey(instance.UserID, instance.ChallengeName)
	expiryKey := utils.InstanceExpiryToKey(instanceID)
	Cache.Set(ctx, expiryKey, instanceID, newTTL)
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

	instanceIDs, err := pruneUserActiveInstancesLocked(ctx, userID)
	if err != nil {
		return 0, err
	}

	return len(instanceIDs), nil
}

func pruneUserActiveInstancesLocked(ctx context.Context, userID string) ([]string, error) {
	activeSetKey := utils.UserActiveInstancesToKey(userID)
	instanceIDs, err := Cache.SMembers(ctx, activeSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user active instances: %w", err)
	}

	active := make([]string, 0, len(instanceIDs))
	for _, instanceID := range instanceIDs {
		instanceExists, err := Cache.Exists(ctx, utils.InstanceToKey(instanceID)).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check instance %s: %w", instanceID, err)
		}
		reservationExists, err := Cache.Exists(ctx, utils.InstanceReservationToKey(instanceID)).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to check instance reservation %s: %w", instanceID, err)
		}

		if instanceExists == 0 && reservationExists == 0 {
			Cache.SRem(ctx, activeSetKey, instanceID)
			continue
		}

		active = append(active, instanceID)
	}

	return active, nil
}

func GetInstanceTTL(instanceID string) (time.Duration, error) {
	if Cache == nil {
		return 0, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	expiryKey := utils.InstanceExpiryToKey(instanceID)
	ttl, err := Cache.TTL(ctx, expiryKey).Result()
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

	key := utils.InstanceToKey(instanceID)

	data, err := Cache.Get(ctx, key).Bytes()
	if err != nil {
		Cache.SRem(ctx, utils.InstancesSetKey, instanceID)
		Cache.SRem(ctx, utils.InstanceDeletionSet, instanceID)
		return fmt.Errorf("instance not found: %w", err)
	}

	var instance Instance
	err = json.Unmarshal(data, &instance)
	if err != nil {
		return fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	instance.State = InstanceStateDeleting
	queuedData, err := json.Marshal(&instance)
	if err != nil {
		return fmt.Errorf("failed to marshal queued instance: %w", err)
	}
	userKey := utils.UserChallengeToKey(instance.UserID, instance.ChallengeName)
	expiryKey := utils.InstanceExpiryToKey(instanceID)
	activeSetKey := utils.UserActiveInstancesToKey(instance.UserID)
	reservationKey := utils.InstanceReservationToKey(instanceID)

	if err := Cache.Eval(ctx, queueInstanceForDeletionScript, []string{
		key,
		utils.InstanceDeletionQueue,
		utils.InstanceDeletionSet,
		utils.InstancesSetKey,
		userKey,
		expiryKey,
		activeSetKey,
		reservationKey,
	}, instanceID, queuedData).Err(); err != nil {
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

	data, err := Cache.RPop(ctx, utils.InstanceDeletionQueue).Bytes()
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

func DeleteInstanceMetadata(instanceID string) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	key := utils.InstanceToKey(instanceID)
	data, err := Cache.Get(ctx, key).Bytes()
	if err != nil {
		Cache.SRem(ctx, utils.InstancesSetKey, instanceID)
		Cache.SRem(ctx, utils.InstanceDeletionSet, instanceID)
		Cache.Del(ctx, utils.InstanceExpiryToKey(instanceID))
		Cache.Del(ctx, utils.InstanceReservationToKey(instanceID))
		return nil
	}

	var instance Instance
	if err := json.Unmarshal(data, &instance); err != nil {
		pipe := Cache.TxPipeline()
		pipe.Del(ctx, key)
		pipe.Del(ctx, utils.InstanceExpiryToKey(instanceID))
		pipe.Del(ctx, utils.InstanceReservationToKey(instanceID))
		pipe.SRem(ctx, utils.InstancesSetKey, instanceID)
		pipe.SRem(ctx, utils.InstanceDeletionSet, instanceID)
		if _, execErr := pipe.Exec(ctx); execErr != nil {
			return fmt.Errorf("failed to delete corrupt instance metadata: %w", execErr)
		}
		return fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	expiryKey := utils.InstanceExpiryToKey(instanceID)
	userKey := utils.UserChallengeToKey(instance.UserID, instance.ChallengeName)
	activeSetKey := utils.UserActiveInstancesToKey(instance.UserID)
	reservationKey := utils.InstanceReservationToKey(instanceID)

	pipe := Cache.TxPipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, expiryKey)
	pipe.Del(ctx, userKey)
	pipe.Del(ctx, reservationKey)
	pipe.SRem(ctx, utils.InstancesSetKey, instanceID)
	pipe.SRem(ctx, utils.InstanceDeletionSet, instanceID)
	pipe.SRem(ctx, activeSetKey, instanceID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete instance metadata: %w", err)
	}

	return nil
}

func RestoreQueuedInstance(instance *Instance) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}
	if instance == nil {
		return fmt.Errorf("instance is nil")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	instance.State = InstanceStateActive
	data, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal instance: %w", err)
	}

	pipe := Cache.TxPipeline()
	pipe.Set(ctx, utils.InstanceToKey(instance.InstanceID), data, 0)
	pipe.SAdd(ctx, utils.InstancesSetKey, instance.InstanceID)
	pipe.SAdd(ctx, utils.UserActiveInstancesToKey(instance.UserID), instance.InstanceID)

	ttl := time.Until(instance.ExpiresAt)
	if ttl <= 0 {
		ttl = time.Second
	}
	pipe.Set(ctx, utils.InstanceExpiryToKey(instance.InstanceID), instance.InstanceID, ttl)
	pipe.Set(ctx, utils.UserChallengeToKey(instance.UserID, instance.ChallengeName), instance.InstanceID, ttl)
	pipe.Del(ctx, utils.InstanceReservationToKey(instance.InstanceID))
	pipe.SRem(ctx, utils.InstanceDeletionSet, instance.InstanceID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to restore queued instance metadata: %w", err)
	}

	return nil
}

func GetDeletionQueueLength() (int64, error) {
	if Cache == nil {
		return 0, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	return Cache.LLen(ctx, utils.InstanceDeletionQueue).Result()
}

func GetExpiredInstances() ([]*Instance, error) {
	if Cache == nil {
		return nil, fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	CacheMutex.Lock()
	defer CacheMutex.Unlock()

	instanceIDs, err := Cache.SMembers(ctx, utils.InstancesSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance IDs: %w", err)
	}

	now := time.Now()
	var expired []*Instance

	for _, id := range instanceIDs {
		key := utils.InstanceToKey(id)
		data, err := Cache.Get(ctx, key).Bytes()
		if err != nil {
			Cache.SRem(ctx, utils.InstancesSetKey, id)
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
