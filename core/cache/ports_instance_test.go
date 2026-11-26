package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/utils"
)

func setupRedisIntegrationTest(t *testing.T) func() {
	t.Helper()

	addr := os.Getenv("BEAST_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set BEAST_TEST_REDIS_ADDR to run Redis cache integration tests")
	}

	db := 0
	if rawDB := os.Getenv("BEAST_TEST_REDIS_DB"); rawDB != "" {
		parsed, err := strconv.Atoi(rawDB)
		if err != nil {
			t.Fatalf("invalid BEAST_TEST_REDIS_DB: %v", err)
		}
		db = parsed
	}

	previousCache := Cache
	previousMutex := CacheMutex
	previousConfig := cacheConfig

	CacheMutex = &sync.Mutex{}
	Cache = redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: os.Getenv("BEAST_TEST_REDIS_USER"),
		Password: os.Getenv("BEAST_TEST_REDIS_PASSWORD"),
		DB:       db,
	})
	cacheConfig.RedisConfig.DB = db

	ctx := context.Background()
	if err := Cache.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	if os.Getenv("BEAST_TEST_REDIS_FLUSH") == "1" {
		if err := Cache.FlushDB(ctx).Err(); err != nil {
			t.Fatalf("flush redis db: %v", err)
		}
	}

	return func() {
		_ = Cache.Close()
		Cache = previousCache
		CacheMutex = previousMutex
		cacheConfig = previousConfig
	}
}

func TestConcurrentMultiPortReservationDoesNotOverlap(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()

	host := fmt.Sprintf("beast-test-host-%d", time.Now().UnixNano())
	defer func() {
		Cache.Del(context.Background(), utils.HostToKey(host))
	}()

	const workers = 40
	const portsPerWorker = 3
	errCh := make(chan error, workers)
	results := make(chan []uint32, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ports, err := GetFreePortsOnHost(host, 30000, 1000, portsPerWorker)
			if err != nil {
				errCh <- err
				return
			}
			results <- ports
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)
	close(results)

	for err := range errCh {
		t.Fatalf("reserve ports: %v", err)
	}

	seen := map[uint32]bool{}
	for ports := range results {
		if len(ports) != portsPerWorker {
			t.Fatalf("expected %d ports per reservation, got %d", portsPerWorker, len(ports))
		}
		for _, port := range ports {
			if seen[port] {
				t.Fatalf("port %d was allocated more than once", port)
			}
			seen[port] = true
		}
	}

	expected := workers * portsPerWorker
	if len(seen) != expected {
		t.Fatalf("expected %d unique reserved ports, got %d", expected, len(seen))
	}
}

func TestAssignAndFreeContainerPortsOnHost(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()

	host := fmt.Sprintf("beast-test-free-host-%d", time.Now().UnixNano())
	owner := fmt.Sprintf("beast-test-owner-%d", time.Now().UnixNano())
	defer func() {
		Cache.Del(context.Background(), utils.HostToKey(host), utils.ContainerToKey(host, owner))
	}()

	ports, err := GetFreePortsOnHost(host, 31000, 10, 2)
	if err != nil {
		t.Fatalf("reserve ports: %v", err)
	}
	if err := AssignPortsOnHostToContainer(host, owner, ports); err != nil {
		t.Fatalf("assign ports: %v", err)
	}

	assigned, err := GetContainerPortsOnHost(host, owner)
	if err != nil {
		t.Fatalf("get assigned ports: %v", err)
	}
	if len(assigned) != len(ports) {
		t.Fatalf("expected %d assigned ports, got %d", len(ports), len(assigned))
	}

	if err := FreeContainerPortsOnHost(host, owner); err != nil {
		t.Fatalf("free assigned ports: %v", err)
	}

	assignedAfterFree, err := GetContainerPortsOnHost(host, owner)
	if err != nil {
		t.Fatalf("get assigned ports after free: %v", err)
	}
	if len(assignedAfterFree) != 0 {
		t.Fatalf("expected no assigned ports after free, got %v", assignedAfterFree)
	}

	reallocated, err := GetFreePortsOnHost(host, 31000, 10, 2)
	if err != nil {
		t.Fatalf("reserve ports after free: %v", err)
	}
	for i := range ports {
		if reallocated[i] != ports[i] {
			t.Fatalf("expected freed port %d to be reusable, got %d", ports[i], reallocated[i])
		}
	}
}

func TestInstanceMetadataOutlivesExpiryMarkerAndQueue(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()
	if os.Getenv("BEAST_TEST_REDIS_FLUSH") != "1" {
		t.Skip("set BEAST_TEST_REDIS_FLUSH=1 for deletion queue tests")
	}

	instanceID := fmt.Sprintf("inst-%d", time.Now().UnixNano())
	instance := &Instance{
		InstanceID:     instanceID,
		ChallengeName:  "durable-instance",
		ContainerID:    "container-" + instanceID,
		PortOwner:      "owner-" + instanceID,
		Port:           []uint32{31337},
		UserID:         "user-1",
		Username:       "user-1",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(50 * time.Millisecond),
		DeploymentType: "standard_docker",
		ServerDeployed: "localhost",
	}

	if err := SaveInstance(instance, 50*time.Millisecond); err != nil {
		t.Fatalf("save instance: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	if _, err := GetInstance(instanceID); err != nil {
		t.Fatalf("durable instance metadata expired with marker: %v", err)
	}

	expired, err := GetExpiredInstances()
	if err != nil {
		t.Fatalf("get expired instances: %v", err)
	}
	if len(expired) != 1 || expired[0].InstanceID != instanceID {
		t.Fatalf("expected durable expired instance %s, got %#v", instanceID, expired)
	}

	if err := QueueInstanceForDeletion(instanceID); err != nil {
		t.Fatalf("queue instance for deletion: %v", err)
	}
	if _, err := GetInstance(instanceID); err != nil {
		t.Fatalf("queueing deletion should keep durable metadata readable: %v", err)
	}

	queued, err := PopInstanceForDeletion()
	if err != nil {
		t.Fatalf("pop queued instance: %v", err)
	}
	if queued == nil || queued.InstanceID != instanceID {
		t.Fatalf("expected queued instance %s, got %#v", instanceID, queued)
	}

	if err := DeleteInstanceMetadata(instanceID); err != nil {
		t.Fatalf("delete instance metadata: %v", err)
	}
	if _, err := GetInstance(instanceID); err == nil {
		t.Fatalf("expected instance metadata to be deleted after successful cleanup")
	}
}

func TestRedisExpiryMarkerSubscription(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()

	if err := EnableKeyspaceExpiryNotifications(); err != nil {
		t.Skipf("redis keyspace notifications are unavailable: %v", err)
	}

	instanceID := fmt.Sprintf("event-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		if err := SubscribeExpiredInstanceMarkers(ctx, func(expiredInstanceID string) {
			events <- expiredInstanceID
		}); err != nil && ctx.Err() == nil {
			errCh <- err
		}
	}()

	time.Sleep(50 * time.Millisecond)
	if err := Cache.Set(context.Background(), utils.InstanceExpiryToKey(instanceID), instanceID, 50*time.Millisecond).Err(); err != nil {
		t.Fatalf("set expiry marker: %v", err)
	}

	select {
	case got := <-events:
		if got != instanceID {
			t.Fatalf("expected expiry event for %s, got %s", instanceID, got)
		}
	case err := <-errCh:
		t.Fatalf("expiry subscriber failed: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for Redis expiry marker event")
	}
}

func TestSpawningReservationPreventsOrphanStatus(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()

	instanceID := fmt.Sprintf("spawn-%d", time.Now().UnixNano())
	defer func() {
		_ = ClearInstanceSpawning(instanceID)
		_ = DeleteInstance(instanceID)
	}()

	known, err := IsInstanceKnown(instanceID)
	if err != nil {
		t.Fatalf("check unknown instance: %v", err)
	}
	if known {
		t.Fatalf("expected unknown instance %s not to be marked known", instanceID)
	}

	if err := MarkInstanceSpawning(instanceID, 5*time.Second); err != nil {
		t.Fatalf("mark instance spawning: %v", err)
	}

	known, err = IsInstanceKnown(instanceID)
	if err != nil {
		t.Fatalf("check spawning instance: %v", err)
	}
	if !known {
		t.Fatalf("expected spawning instance %s to be marked known", instanceID)
	}

	instance := &Instance{
		InstanceID:     instanceID,
		ChallengeName:  "spawning-test",
		ContainerID:    "container-" + instanceID,
		PortOwner:      "owner-" + instanceID,
		Port:           []uint32{31338},
		UserID:         "user-spawn",
		Username:       "user-spawn",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(time.Minute),
		DeploymentType: "docker_compose",
		ServerDeployed: "localhost",
	}
	if err := SaveInstance(instance, time.Minute); err != nil {
		t.Fatalf("save instance: %v", err)
	}

	exists, err := Cache.Exists(context.Background(), utils.InstanceSpawningToKey(instanceID)).Result()
	if err != nil {
		t.Fatalf("check spawning key after save: %v", err)
	}
	if exists != 0 {
		t.Fatalf("expected spawning marker to be removed on SaveInstance")
	}

	known, err = IsInstanceKnown(instanceID)
	if err != nil {
		t.Fatalf("check saved instance: %v", err)
	}
	if !known {
		t.Fatalf("expected saved instance %s to remain known", instanceID)
	}

	if err := DeleteInstance(instanceID); err != nil {
		t.Fatalf("delete saved instance: %v", err)
	}

	known, err = IsInstanceKnown(instanceID)
	if err != nil {
		t.Fatalf("check deleted instance: %v", err)
	}
	if known {
		t.Fatalf("expected deleted instance %s not to be marked known", instanceID)
	}
}
