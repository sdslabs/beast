package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

func TestInstanceActiveAtDefaultsMissingStateToActive(t *testing.T) {
	now := time.Now()
	instance := &Instance{ExpiresAt: now.Add(time.Minute)}
	if instance.StateOrActive() != InstanceStateActive {
		t.Fatalf("expected missing state to default to active, got %q", instance.StateOrActive())
	}
	if !instance.ActiveAt(now) {
		t.Fatalf("expected unexpired missing-state instance to be active")
	}

	instance.State = InstanceStateDeleting
	if instance.ActiveAt(now) {
		t.Fatalf("expected deleting instance to be inactive")
	}

	instance.State = InstanceStateActive
	instance.ExpiresAt = now.Add(-time.Second)
	if instance.ActiveAt(now) {
		t.Fatalf("expected expired instance to be inactive")
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
		Port:           31337,
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
	deleting, err := GetInstance(instanceID)
	if err != nil {
		t.Fatalf("queueing deletion should keep durable metadata readable: %v", err)
	}
	if deleting.StateOrActive() != InstanceStateDeleting {
		t.Fatalf("expected queued instance state %q, got %q", InstanceStateDeleting, deleting.StateOrActive())
	}
	count, err := CountUserInstances(instance.UserID)
	if err != nil {
		t.Fatalf("count after queue: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected queued instance to keep user slot, got %d", count)
	}
	if userInstance, err := GetUserInstance(instance.UserID, instance.ChallengeName); err != nil || userInstance.InstanceID != instanceID {
		t.Fatalf("expected queued instance to keep user mapping, got %#v err=%v", userInstance, err)
	}
	queueLen, err := GetDeletionQueueLength()
	if err != nil {
		t.Fatalf("get deletion queue length: %v", err)
	}
	if queueLen != 1 {
		t.Fatalf("expected one queued deletion, got %d", queueLen)
	}
	if err := QueueInstanceForDeletion(instanceID); err != nil {
		t.Fatalf("queue instance for deletion twice: %v", err)
	}
	queueLen, err = GetDeletionQueueLength()
	if err != nil {
		t.Fatalf("get deletion queue length after duplicate queue: %v", err)
	}
	if queueLen != 1 {
		t.Fatalf("expected duplicate queue to be deduped, got %d", queueLen)
	}

	queued, err := PopInstanceForDeletion()
	if err != nil {
		t.Fatalf("pop queued instance: %v", err)
	}
	if queued == nil || queued.InstanceID != instanceID {
		t.Fatalf("expected queued instance %s, got %#v", instanceID, queued)
	}
	if queued.StateOrActive() != InstanceStateDeleting {
		t.Fatalf("expected popped instance state %q, got %q", InstanceStateDeleting, queued.StateOrActive())
	}
	if err := QueueInstanceForDeletion(instanceID); err != nil {
		t.Fatalf("requeue popped instance after simulated worker crash: %v", err)
	}
	queueLen, err = GetDeletionQueueLength()
	if err != nil {
		t.Fatalf("get deletion queue length after crash requeue: %v", err)
	}
	if queueLen != 1 {
		t.Fatalf("expected popped instance to be requeueable, got %d", queueLen)
	}
	queued, err = PopInstanceForDeletion()
	if err != nil {
		t.Fatalf("pop requeued instance: %v", err)
	}
	if queued == nil || queued.InstanceID != instanceID {
		t.Fatalf("expected requeued instance %s, got %#v", instanceID, queued)
	}

	if err := DeleteInstanceMetadata(instanceID); err != nil {
		t.Fatalf("delete instance metadata: %v", err)
	}
	if err := DeleteInstanceMetadata(instanceID); err != nil {
		t.Fatalf("delete instance metadata should be idempotent: %v", err)
	}
	if _, err := GetInstance(instanceID); err == nil {
		t.Fatalf("expected instance metadata to be deleted after successful cleanup")
	}
	count, err = CountUserInstances(instance.UserID)
	if err != nil {
		t.Fatalf("count after metadata delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected deleted instance to free user slot, got %d", count)
	}
}

func TestConcurrentInstanceReservationSameChallengeCreatesOneActiveSlot(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()

	userID := fmt.Sprintf("reservation-user-%d", time.Now().UnixNano())
	challengeName := "reservation-same-challenge"
	const workers = 64

	errCh := make(chan error, workers)
	grantedIDs := make(chan string, workers)
	activeCount := int32(0)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			instanceID := fmt.Sprintf("reservation-%d", i)
			result, err := ReserveInstanceSlot(userID, challengeName, instanceID, time.Minute, 10)
			if err != nil {
				errCh <- err
				return
			}
			switch result.Status {
			case InstanceReservationGranted:
				grantedIDs <- instanceID
			case InstanceReservationChallengeActive:
				atomic.AddInt32(&activeCount, 1)
			default:
				errCh <- fmt.Errorf("unexpected reservation status %v", result.Status)
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(errCh)
	close(grantedIDs)

	for err := range errCh {
		t.Fatalf("reserve same challenge: %v", err)
	}

	var granted []string
	for instanceID := range grantedIDs {
		granted = append(granted, instanceID)
	}
	if len(granted) != 1 {
		t.Fatalf("expected exactly one granted reservation, got %d", len(granted))
	}
	if activeCount != workers-1 {
		t.Fatalf("expected %d active-conflict reservations, got %d", workers-1, activeCount)
	}

	count, err := CountUserInstances(userID)
	if err != nil {
		t.Fatalf("count user instances: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one active reserved slot, got %d", count)
	}

	if err := ReleaseInstanceReservation(userID, challengeName, granted[0]); err != nil {
		t.Fatalf("release reservation: %v", err)
	}
}

func TestConcurrentInstanceReservationRespectsUserLimit(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()

	userID := fmt.Sprintf("limit-user-%d", time.Now().UnixNano())
	const workers = 64
	const maxInstances = 3

	errCh := make(chan error, workers)
	grantedIDs := make(chan string, workers)
	start := make(chan struct{})
	var limitReached int32
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			instanceID := fmt.Sprintf("limit-instance-%d", i)
			challengeName := fmt.Sprintf("limit-challenge-%d", i)
			result, err := ReserveInstanceSlot(userID, challengeName, instanceID, time.Minute, maxInstances)
			if err != nil {
				errCh <- err
				return
			}
			switch result.Status {
			case InstanceReservationGranted:
				grantedIDs <- instanceID + ":" + challengeName
			case InstanceReservationLimitReached:
				atomic.AddInt32(&limitReached, 1)
			default:
				errCh <- fmt.Errorf("unexpected reservation status %v", result.Status)
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(errCh)
	close(grantedIDs)

	for err := range errCh {
		t.Fatalf("reserve across challenges: %v", err)
	}

	var granted []string
	for item := range grantedIDs {
		granted = append(granted, item)
	}
	if len(granted) != maxInstances {
		t.Fatalf("expected %d granted reservations, got %d", maxInstances, len(granted))
	}
	if limitReached != workers-maxInstances {
		t.Fatalf("expected %d limit responses, got %d", workers-maxInstances, limitReached)
	}

	count, err := CountUserInstances(userID)
	if err != nil {
		t.Fatalf("count user instances: %v", err)
	}
	if count != maxInstances {
		t.Fatalf("expected %d active reserved slots, got %d", maxInstances, count)
	}

	for _, item := range granted {
		parts := strings.Split(item, ":")
		if err := ReleaseInstanceReservation(userID, parts[1], parts[0]); err != nil {
			t.Fatalf("release reservation %s: %v", item, err)
		}
	}
}

func TestReleaseInstanceReservationFreesUserSlot(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()

	userID := fmt.Sprintf("release-user-%d", time.Now().UnixNano())
	first, err := ReserveInstanceSlot(userID, "release-one", "release-instance-one", time.Minute, 1)
	if err != nil {
		t.Fatalf("reserve first slot: %v", err)
	}
	if first.Status != InstanceReservationGranted {
		t.Fatalf("expected first reservation granted, got %v", first.Status)
	}

	limited, err := ReserveInstanceSlot(userID, "release-two", "release-instance-two", time.Minute, 1)
	if err != nil {
		t.Fatalf("reserve over limit: %v", err)
	}
	if limited.Status != InstanceReservationLimitReached {
		t.Fatalf("expected limit before release, got %v", limited.Status)
	}

	if err := ReleaseInstanceReservation(userID, "release-one", "release-instance-one"); err != nil {
		t.Fatalf("release first slot: %v", err)
	}

	second, err := ReserveInstanceSlot(userID, "release-two", "release-instance-two", time.Minute, 1)
	if err != nil {
		t.Fatalf("reserve after release: %v", err)
	}
	if second.Status != InstanceReservationGranted {
		t.Fatalf("expected reservation after release, got %v", second.Status)
	}
}

func TestRestoreQueuedInstanceRestoresUserMappings(t *testing.T) {
	cleanup := setupRedisIntegrationTest(t)
	defer cleanup()
	if os.Getenv("BEAST_TEST_REDIS_FLUSH") != "1" {
		t.Skip("set BEAST_TEST_REDIS_FLUSH=1 for deletion queue tests")
	}

	instanceID := fmt.Sprintf("restore-%d", time.Now().UnixNano())
	instance := &Instance{
		InstanceID:     instanceID,
		ChallengeName:  "restore-challenge",
		ContainerID:    "container-" + instanceID,
		PortOwner:      "owner-" + instanceID,
		Port:           31338,
		UserID:         "restore-user",
		Username:       "restore-user",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(time.Minute),
		DeploymentType: "standard_docker",
		ServerDeployed: "localhost",
	}

	if err := SaveInstance(instance, time.Minute); err != nil {
		t.Fatalf("save instance: %v", err)
	}
	if err := QueueInstanceForDeletion(instanceID); err != nil {
		t.Fatalf("queue instance: %v", err)
	}
	count, err := CountUserInstances(instance.UserID)
	if err != nil {
		t.Fatalf("count after queue: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected queued instance to keep active slot, got %d", count)
	}
	deleting, err := GetUserInstance(instance.UserID, instance.ChallengeName)
	if err != nil {
		t.Fatalf("get deleting user instance: %v", err)
	}
	if deleting.StateOrActive() != InstanceStateDeleting {
		t.Fatalf("expected queued user instance state %q, got %q", InstanceStateDeleting, deleting.StateOrActive())
	}
	sameChallenge, err := ReserveInstanceSlot(instance.UserID, instance.ChallengeName, "restore-same-challenge", time.Minute, 10)
	if err != nil {
		t.Fatalf("reserve same challenge while deleting: %v", err)
	}
	if sameChallenge.Status != InstanceReservationChallengeActive || sameChallenge.ExistingInstanceID != instanceID {
		t.Fatalf("expected deleting instance to keep challenge active, got %#v", sameChallenge)
	}
	limited, err := ReserveInstanceSlot(instance.UserID, "restore-other-challenge", "restore-other", time.Minute, 1)
	if err != nil {
		t.Fatalf("reserve other challenge while deleting: %v", err)
	}
	if limited.Status != InstanceReservationLimitReached {
		t.Fatalf("expected deleting instance to keep user slot occupied, got %#v", limited)
	}

	queued, err := PopInstanceForDeletion()
	if err != nil {
		t.Fatalf("pop queued instance: %v", err)
	}
	if err := RestoreQueuedInstance(queued); err != nil {
		t.Fatalf("restore queued instance: %v", err)
	}

	restored, err := GetUserInstance(instance.UserID, instance.ChallengeName)
	if err != nil {
		t.Fatalf("get restored user instance: %v", err)
	}
	if restored.InstanceID != instanceID {
		t.Fatalf("expected restored instance %s, got %s", instanceID, restored.InstanceID)
	}
	if restored.StateOrActive() != InstanceStateActive {
		t.Fatalf("expected restored instance state %q, got %q", InstanceStateActive, restored.StateOrActive())
	}
	count, err = CountUserInstances(instance.UserID)
	if err != nil {
		t.Fatalf("count after restore: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected restored instance in active set, got %d", count)
	}
	ttl, err := GetInstanceTTL(instanceID)
	if err != nil {
		t.Fatalf("get restored ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("expected restored expiry marker ttl > 0, got %v", ttl)
	}
	if err := QueueInstanceForDeletion(instanceID); err != nil {
		t.Fatalf("queue restored instance again: %v", err)
	}
	queueLen, err := GetDeletionQueueLength()
	if err != nil {
		t.Fatalf("get deletion queue length after restored retry: %v", err)
	}
	if queueLen != 1 {
		t.Fatalf("expected restored instance to be requeueable, got queue length %d", queueLen)
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
