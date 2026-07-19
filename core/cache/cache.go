package cache

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

var (
	CacheMutex *sync.Mutex
	Cache      *redis.Client
)

var cacheConfig RedisConfig

type RedisConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DB       uint32
}

func Configure(user, password, host, port string, db uint32) {
	cacheConfig = RedisConfig{User: user, Password: password, Host: host, Port: port, DB: db}
}

func LoadCacheConfig() error {
	if cacheConfig.Host == "" || cacheConfig.Port == "" || cacheConfig.Password == "" {
		return fmt.Errorf("cache is not configured")
	}
	return nil
}

func redisAddress(config RedisConfig) string {
	return net.JoinHostPort(config.Host, config.Port)
}

// Connect redis
func ConnectCache() error {
	if err := LoadCacheConfig(); err != nil {
		return err
	}
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddress(cacheConfig),
		Username: cacheConfig.User,
		Password: cacheConfig.Password,
		DB:       int(cacheConfig.DB),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("connect to Redis: %w", err)
	}

	Cache = client
	log.Debug("Cache initialized")
	return nil
}

// Set up the initial bootstrapping for interacting with the
// Postgresql database for beast. The Db variable is the connection variable for the
// database, which is not closed after creating a connection here and can
// be used further after this.
func Init() error {
	CacheMutex = &sync.Mutex{}
	if Cache == nil {
		if err := ConnectCache(); err != nil {
			return fmt.Errorf("initialize cache: %w", err)
		}
	}
	return nil
}

func EnableKeyspaceExpiryNotifications() error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}

	ctx := context.Background()
	result, err := Cache.Do(ctx, "CONFIG", "GET", "notify-keyspace-events").Result()
	if err != nil {
		return err
	}

	current := ""
	if values, ok := result.([]interface{}); ok && len(values) >= 2 {
		current = fmt.Sprint(values[1])
	}

	next := current
	if !strings.Contains(next, "E") {
		next += "E"
	}
	if !strings.Contains(next, "x") {
		next += "x"
	}

	if next == current {
		return nil
	}

	return Cache.Do(ctx, "CONFIG", "SET", "notify-keyspace-events", next).Err()
}

func SubscribeExpiredInstanceMarkers(ctx context.Context, handler func(instanceID string)) error {
	if Cache == nil {
		return fmt.Errorf("redis cache not initialized")
	}

	pattern := fmt.Sprintf("__keyevent@%d__:expired", cacheConfig.DB)
	pubsub := Cache.PSubscribe(ctx, pattern)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}

			instanceID, ok := utils.InstanceIDFromExpiryKey(msg.Payload)
			if !ok {
				continue
			}
			handler(instanceID)
		}
	}
}

func Close() error {
	if Cache == nil {
		log.Warnln(fmt.Sprintf("Trying to close database connection when no connection is established..."))
		return nil
	}

	err := Cache.Close()
	if err != nil {
		log.Errorln(fmt.Sprintf("Error while closing cache connection gracefully: %s, attempting to terminate forcefully", err.Error()))
		return TerminateCacheConnections()
	}

	return nil
}

func BackupAndReset() {
	if err := LoadCacheConfig(); err != nil {
		log.Error(err)
		return
	}

	err := BackupCache()
	if err != nil {
		log.Errorf("Error while backing up cache: %s", err)
		return
	}
	err = ResetCache()
	if err != nil {
		log.Errorf("Error while resetting up cache: %s", err)
		return
	}

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_REMOTES_DIR)
	err = utils.CreateIfNotExistDir(backupPath)
	if err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return
	}

	backupPath = filepath.Join(backupPath, core.BEAST_REMOTES_DIR+time.Now().Format("20060102150405")+".bak")
	oldPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	err = os.Rename(oldPath, backupPath)
	if err != nil {
		log.Errorf("Error while backing up remote dir: %s", err)
		return
	}

	backupPath = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_STAGING_DIR)

	err = utils.CreateIfNotExistDir(backupPath)
	if err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return
	}

	oldPath = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	backupPath = filepath.Join(backupPath, core.BEAST_STAGING_DIR+time.Now().Format("20060102150405")+".bak")
	err = os.Rename(oldPath, backupPath)
	if err != nil {
		log.Errorf("Error while backing up staging dir: %s", err)
		return
	}
}

func BackupCache() error {
	if cacheConfig == (RedisConfig{}) {
		if err := LoadCacheConfig(); err != nil {
			return err
		}
	}

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_CACHE_DIR)
	err := utils.CreateIfNotExistDir(backupPath)
	if err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return err
	}

	backupFile := fmt.Sprintf("%d_%s.bak", cacheConfig.DB, time.Now().Format("20060102150405"))

	args := []string{
		"-h", cacheConfig.Host,
		"-p", cacheConfig.Port,
		"-n", strconv.FormatUint(uint64(cacheConfig.DB), 10),
		"--rdb", filepath.Join(backupPath, backupFile),
	}
	if cacheConfig.User != "" {
		args = append(args, "--user", cacheConfig.User)
	}

	cmd := exec.Command("redis-cli", args...)

	cmd.Env = append(os.Environ(), fmt.Sprintf("REDISCLI_AUTH=%s", cacheConfig.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Backup error: %s\n", string(output))
		return err
	}
	log.Debug("Backup successful.")
	return nil
}

func ResetCache() error {
	if cacheConfig == (RedisConfig{}) {
		if err := LoadCacheConfig(); err != nil {
			return err
		}
	}
	err := TerminateCacheConnections()
	if err != nil {
		log.Errorf("Unable to terminate connections %s", err)
		return err
	}

	dropCmd := exec.Command(
		"redis-cli",
		"-h", cacheConfig.Host,
		"-p", cacheConfig.Port,
		"--user", cacheConfig.User,
		"-n", strconv.FormatUint(uint64(cacheConfig.DB), 10),
		"FLUSHDB",
	)

	dropCmd.Env = append(os.Environ(), fmt.Sprintf("REDISCLI_AUTH=%s", cacheConfig.Password))

	output, err := dropCmd.CombinedOutput()
	if err != nil {
		log.Printf("Drop Cache error: %s\n", string(output))
		return err
	}

	log.Debug("Reset successful.")
	return nil
}

// Terminate all active connections before dropping
func TerminateCacheConnections() error {
	if cacheConfig == (RedisConfig{}) {
		if err := LoadCacheConfig(); err != nil {
			return err
		}
	}

	cache := redis.NewClient(&redis.Options{
		Addr:     redisAddress(cacheConfig),
		Username: core.REDIS_DEFAULT_USER,
		Password: utils.PromptSecret("Enter default redis user password"),
	})

	_, err := cache.Ping(context.Background()).Result()
	if err != nil {
		log.Errorf("Terminate connections error: %s\n", err.Error())
	}

	defer cache.Close()

	_, err = cache.Do(context.Background(),
		"CLIENT", "KILL",
		"USER", cacheConfig.User,
		"SKIPME", "yes",
	).Result()

	if err != nil {
		log.Errorf("Terminate connections error: %s\n", err.Error())
	}

	return nil
}

func RestoreCache(backupFile string) error {
	/*
		The primary issue with restoring cache is that it needs to be written to /var/lib and redis needs to be restarted.
		Redis will then pick up the changes and continue from there.
	*/
	return nil
}
