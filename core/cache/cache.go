package cache

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	User     string `toml:"user"`
	Password string `toml:"password"`
	Host     string `toml:"host"`
	Port     string `toml:"port"`
	DB       int    `toml:"db"`
}

// Db config is loaded separately here for temp use because init() function is
// called during initialization of package.
// It is also loaded during db backup/reset
func LoadCacheConfig() {
	if _, err := toml.DecodeFile(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME), &cacheConfig); err != nil {
		log.Fatalf("Error loading TOML file: %v", err)
	}
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
	LoadCacheConfig()

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
	if cacheConfig == (Config{}) {
		LoadCacheConfig()
	}

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_CACHE_DIR)
	err := utils.CreateIfNotExistDir(backupPath)
	if err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return err
	}

	backupFile := fmt.Sprintf("%d_%s.bak", cacheConfig.RedisConfig.DB, time.Now().Format("20060102150405"))

	args := []string{
		"-h", cacheConfig.RedisConfig.Host,
		"-p", cacheConfig.RedisConfig.Port,
		"-n", strconv.Itoa(cacheConfig.RedisConfig.DB),
		"--rdb", filepath.Join(backupPath, backupFile),
	}
	if cacheConfig.RedisConfig.User != "" {
		args = append(args, "--user", cacheConfig.RedisConfig.User)
	}
	if cacheConfig.RedisConfig.Password != "" {
		args = append(args, "--pass", cacheConfig.RedisConfig.Password)
	}

	cmd := exec.Command("redis-cli", args...)

	cmd.Env = append(os.Environ(), fmt.Sprintf("REDISCLI_AUTH=%s", cacheConfig.RedisConfig.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Backup error: %s\n", string(output))
		return err
	}
	log.Debug("Backup successful.")
	return nil
}

func ResetCache() error {
	if cacheConfig == (Config{}) {
		LoadCacheConfig()
	}
	err := TerminateCacheConnections()
	if err != nil {
		log.Errorf("Unable to terminate connections %s", err)
		return err
	}

	dropCmd := exec.Command(
		"redis-cli",
		"-h", cacheConfig.RedisConfig.Host,
		"-p", cacheConfig.RedisConfig.Port,
		"--user", cacheConfig.RedisConfig.User,
		"-n", strconv.Itoa(cacheConfig.RedisConfig.DB),
		"FLUSHDB",
	)

	dropCmd.Env = append(os.Environ(), fmt.Sprintf("REDISCLI_AUTH=%s", cacheConfig.RedisConfig.Password))

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
	if cacheConfig == (Config{}) {
		LoadCacheConfig()
	}

	cache := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cacheConfig.RedisConfig.Host, cacheConfig.RedisConfig.Port),
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
		"USER", cacheConfig.RedisConfig.User,
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
