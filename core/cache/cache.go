package cache

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

var (
	CacheMutex *sync.Mutex
	Cache      *redis.Client
	cacheError error
)

var (
	cacheConfig Config
)

type Config struct {
	RedisConfig RedisConfig `toml:"redis_config"`
}
type RedisConfig struct {
	User            string `toml:"user"`
	Password        string `toml:"password"`
	DefaultPassword string `toml:"default_password"`
	Host            string `toml:"host"`
	Port            string `toml:"port"`
	DB              int    `toml:"db"`
}

func (rc *RedisConfig) applyEnvOverrides() {
	utils.LoadEnvFile()
	if val := os.Getenv("REDIS_USER"); val != "" {
		rc.User = val
	}
	if val := os.Getenv("REDIS_PASSWORD"); val != "" {
		rc.Password = val
	}
	if val := os.Getenv("REDIS_DEFAULT_PASSWORD"); val != "" {
		rc.DefaultPassword = val
	}
	if val := os.Getenv("REDIS_HOST"); val != "" {
		rc.Host = val
	}
	if val := os.Getenv("REDIS_PORT"); val != "" {
		rc.Port = val
	}
	if val := os.Getenv("REDIS_DB"); val != "" {
		if dbVal, err := strconv.Atoi(val); err == nil {
			rc.DB = dbVal
		}
	}
}

// Db config is loaded separately here for temp use because init() function is
// called during initialization of package.
// It is also loaded during db backup/reset
func LoadCacheConfig() {
	if _, err := toml.DecodeFile(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME), &cacheConfig); err != nil {
		log.Fatalf("Error loading TOML file: %v", err)
	}
	cacheConfig.RedisConfig.applyEnvOverrides()
}

// Connect redis
func ConnectCache() error {
	LoadCacheConfig()
	Cache = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cacheConfig.RedisConfig.Host, cacheConfig.RedisConfig.Port),
		Username: cacheConfig.RedisConfig.User,
		Password: cacheConfig.RedisConfig.Password,
		DB:       cacheConfig.RedisConfig.DB,
	})

	_, err := Cache.Ping(context.Background()).Result()
	if err != nil {
		return fmt.Errorf("failed to connected to redis: %s", err.Error())
	}

	log.Debug("Cache initialized")
	return nil
}

// Set up the initial bootstrapping for interacting with the
// Postgresql database for beast. The Db variable is the connection variable for the
// database, which is not closed after creating a connection here and can
// be used further after this.
func Init() {
	CacheMutex = &sync.Mutex{}
	if Cache == nil {
		cacheError = ConnectCache()
		if cacheError != nil {
			log.Errorf("Error while initializing cache: %s", cacheError.Error())
		}
	}
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

	pattern := fmt.Sprintf("__keyevent@%d__:expired", cacheConfig.RedisConfig.DB)
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

type cacheBackupEntry struct {
	Key   string `json:"key"`
	TTLMs int64  `json:"ttl_ms"`
	Dump  string `json:"dump"`
}

type cacheBackupPayload struct {
	Version   int                `json:"version"`
	DB        int                `json:"db"`
	CreatedAt string             `json:"created_at"`
	Entries   []cacheBackupEntry `json:"entries"`
}

func newConfiguredRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cacheConfig.RedisConfig.Host, cacheConfig.RedisConfig.Port),
		Username: cacheConfig.RedisConfig.User,
		Password: cacheConfig.RedisConfig.Password,
		DB:       cacheConfig.RedisConfig.DB,
	})
}

func dumpCacheEntries(ctx context.Context, rdb *redis.Client) (*cacheBackupPayload, error) {
	payload := &cacheBackupPayload{
		Version:   1,
		DB:        cacheConfig.RedisConfig.DB,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Entries:   []cacheBackupEntry{},
	}

	var cursor uint64
	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, "*", 500).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			dumped, err := rdb.Dump(ctx, key).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("dump key %q: %w", key, err)
			}
			pttl, err := rdb.PTTL(ctx, key).Result()
			if err != nil {
				return nil, fmt.Errorf("pttl key %q: %w", key, err)
			}
			var ttlMs int64
			if pttl > 0 {
				ttlMs = pttl.Milliseconds()
			}
			payload.Entries = append(payload.Entries, cacheBackupEntry{
				Key:   key,
				TTLMs: ttlMs,
				Dump:  base64.StdEncoding.EncodeToString([]byte(dumped)),
			})
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return payload, nil
}

func BackupAndReset() error {
	LoadCacheConfig()

	err := BackupCache()
	if err != nil {
		log.Errorf("Error while backing up cache: %s", err)
		return err
	}
	err = ResetCache()
	if err != nil {
		log.Errorf("Error while resetting up cache: %s", err)
		return err
	}

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_REMOTES_DIR)
	if err = utils.CreateIfNotExistDir(backupPath); err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return err
	}

	oldPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	if _, statErr := os.Stat(oldPath); statErr == nil {
		remoteBackup := filepath.Join(backupPath, core.BEAST_REMOTES_DIR+time.Now().Format("20060102150405")+".bak")
		if err = os.Rename(oldPath, remoteBackup); err != nil {
			log.Errorf("Error while backing up remote dir: %s", err)
			return err
		}
	}

	backupPath = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_STAGING_DIR)
	if err = utils.CreateIfNotExistDir(backupPath); err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return err
	}

	oldPath = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	if _, statErr := os.Stat(oldPath); statErr == nil {
		stagingBackup := filepath.Join(backupPath, core.BEAST_STAGING_DIR+time.Now().Format("20060102150405")+".bak")
		if err = os.Rename(oldPath, stagingBackup); err != nil {
			log.Errorf("Error while backing up staging dir: %s", err)
			return err
		}
	}

	return nil
}

func BackupCache() error {
	if cacheConfig == (Config{}) {
		LoadCacheConfig()
	}

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_CACHE_DIR)
	if err := utils.CreateIfNotExistDir(backupPath); err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return err
	}

	backupFile := fmt.Sprintf("%d_%s.bak", cacheConfig.RedisConfig.DB, time.Now().Format("20060102150405"))
	fullBackupPath := filepath.Join(backupPath, backupFile)

	ctx := context.Background()
	rdb := newConfiguredRedisClient()
	defer rdb.Close()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("failed to connect to redis for backup: %w", err)
	}

	payload, err := dumpCacheEntries(ctx, rdb)
	if err != nil {
		return fmt.Errorf("failed to dump redis keys: %w", err)
	}
	jsonBytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize redis backup: %w", err)
	}

	if _, lookErr := exec.LookPath("redis-cli"); lookErr == nil {
		args := []string{
			"-h", cacheConfig.RedisConfig.Host,
			"-p", cacheConfig.RedisConfig.Port,
			"-n", strconv.Itoa(cacheConfig.RedisConfig.DB),
		}
		if cacheConfig.RedisConfig.User != "" {
			args = append(args, "--user", cacheConfig.RedisConfig.User)
		}
		args = append(args, "--rdb", fullBackupPath)

		cmd := exec.Command("redis-cli", args...)
		if cacheConfig.RedisConfig.Password != "" {
			cmd.Env = append(os.Environ(), fmt.Sprintf("REDISCLI_AUTH=%s", cacheConfig.RedisConfig.Password))
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Warnf("redis-cli RDB backup failed (%v: %s), writing portable JSON backup", err, strings.TrimSpace(string(output)))
			if writeErr := os.WriteFile(fullBackupPath, jsonBytes, 0600); writeErr != nil {
				return writeErr
			}
		} else {
			if writeErr := os.WriteFile(fullBackupPath+".json", jsonBytes, 0600); writeErr != nil {
				return writeErr
			}
		}
	} else {
		if writeErr := os.WriteFile(fullBackupPath, jsonBytes, 0600); writeErr != nil {
			return writeErr
		}
	}

	log.Infof("Cache backup written to %s", fullBackupPath)
	return nil
}

func ResetCache() error {
	if cacheConfig == (Config{}) {
		LoadCacheConfig()
	}
	if err := TerminateCacheConnections(); err != nil {
		log.Warnf("Unable to terminate connections cleanly: %s", err)
	}

	ctx := context.Background()
	rdb := newConfiguredRedisClient()
	defer rdb.Close()

	if err := rdb.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("drop cache error: %w", err)
	}

	log.Debug("Reset successful.")
	return nil
}

// Terminate all active connections before dropping
func TerminateCacheConnections() error {
	if cacheConfig == (Config{}) {
		LoadCacheConfig()
	}

	adminPassword := cacheConfig.RedisConfig.DefaultPassword
	adminUser := core.REDIS_DEFAULT_USER
	if adminPassword == "" {
		adminUser = cacheConfig.RedisConfig.User
		adminPassword = cacheConfig.RedisConfig.Password
	}

	adminClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cacheConfig.RedisConfig.Host, cacheConfig.RedisConfig.Port),
		Username: adminUser,
		Password: adminPassword,
		DB:       cacheConfig.RedisConfig.DB,
	})
	defer adminClient.Close()

	ctx := context.Background()
	if _, err := adminClient.Ping(ctx).Result(); err != nil {
		log.Warnf("Terminate connections ping warning: %s", err.Error())
		return nil
	}

	args := []interface{}{"CLIENT", "KILL", "SKIPME", "yes"}
	if cacheConfig.RedisConfig.User != "" {
		args = []interface{}{"CLIENT", "KILL", "USER", cacheConfig.RedisConfig.User, "SKIPME", "yes"}
	}
	if _, err := adminClient.Do(ctx, args...).Result(); err != nil {
		log.Warnf("Terminate connections warning: %s", err.Error())
	}

	return nil
}

func RestoreCache(backupFile string) error {
	if cacheConfig == (Config{}) {
		LoadCacheConfig()
	}
	if err := utils.ValidateFileExists(backupFile); err != nil {
		return fmt.Errorf("backup file %q not accessible: %w", backupFile, err)
	}

	raw, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("read backup file %q: %w", backupFile, err)
	}

	// If given an RDB file, look for the portable JSON sidecar produced by BackupCache.
	if len(raw) >= 5 && string(raw[:5]) == "REDIS" {
		sidecar := backupFile + ".json"
		if sidecarBytes, sidecarErr := os.ReadFile(sidecar); sidecarErr == nil {
			raw = sidecarBytes
		} else {
			return fmt.Errorf("raw RDB file %q cannot be hot-loaded into running Redis without a %q key dump or Redis server restart", backupFile, sidecar)
		}
	}

	var payload cacheBackupPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("parse cache backup %q: %w", backupFile, err)
	}

	ctx := context.Background()
	rdb := newConfiguredRedisClient()
	defer rdb.Close()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("failed to connect to redis for restore: %w", err)
	}

	for _, entry := range payload.Entries {
		dumped, err := base64.StdEncoding.DecodeString(entry.Dump)
		if err != nil {
			return fmt.Errorf("decode dump for key %q: %w", entry.Key, err)
		}
		ttl := time.Duration(0)
		if entry.TTLMs > 0 {
			ttl = time.Duration(entry.TTLMs) * time.Millisecond
		}
		if err := rdb.RestoreReplace(ctx, entry.Key, ttl, string(dumped)).Err(); err != nil {
			return fmt.Errorf("restore key %q: %w", entry.Key, err)
		}
	}

	log.Infof("Restored %d cache keys from %s", len(payload.Entries), backupFile)
	return nil
}
