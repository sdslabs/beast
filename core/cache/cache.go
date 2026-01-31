package cache

import (
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
	BEAST_GLOBAL_DIR string = filepath.Join(os.Getenv("HOME"), ".beast")
	cacheConfig      Config
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
func ConnectRedis() error {
	LoadCacheConfig()
	Cache = redis.NewClient(&redis.Options{
		Addr:     cacheConfig.RedisConfig.Host + ":" + cacheConfig.RedisConfig.Port,
		Username: cacheConfig.RedisConfig.User,
		Password: cacheConfig.RedisConfig.Password,
		DB:       cacheConfig.RedisConfig.DB,
	})
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
		cacheError = ConnectRedis()
		if cacheError != nil {
			log.Error("Error while initializing the database.", cacheError)
		}
	}
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

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, "backup", core.BEAST_REMOTES_DIR)
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

	backupPath = filepath.Join(core.BEAST_GLOBAL_DIR, "backup", core.BEAST_STAGING_DIR)

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

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, "backup", "cache")
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
	terminateCmd := exec.Command(
		"redis-cli",
		"-h", cacheConfig.RedisConfig.Host,
		"-p", cacheConfig.RedisConfig.Port,
		"--user", cacheConfig.RedisConfig.User,
		"-n", strconv.Itoa(cacheConfig.RedisConfig.DB),
		"CLIENT", "KILL", "USER", cacheConfig.RedisConfig.User,
	)

	terminateCmd.Env = append(os.Environ(), fmt.Sprintf("REDISCLI_AUTH=%s", cacheConfig.RedisConfig.Password))

	output, err := terminateCmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		log.Errorf("Terminate connections error: %s\n", outputStr)
		return err
	}
	log.Debug(outputStr)
	return nil
}

func RestoreCache(backupFile string) error {
	LoadCacheConfig()

	err := TerminateCacheConnections()
	if err != nil {
		log.Errorf("Unable to terminate connections: %s ", err)
		return err
	}

	err = utils.ValidateFileExists(backupFile)
	if err != nil {
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	// TODO: figure out how to do this
	//restoreCmd := exec.Command(
	//	"pg_restore",
	//	"-U", dbConfig.PsqlConf.User,
	//	"-h", dbConfig.PsqlConf.Host,
	//	"-p", dbConfig.PsqlConf.Port,
	//	"-d", dbConfig.PsqlConf.Dbname,
	//	"--no-owner",
	//	"--clean",
	//	"--if-exists",
	//	backupFile,
	//)
	//restoreCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.PsqlConf.Password))
	//
	//output, err := restoreCmd.CombinedOutput()
	//if err != nil {
	//	log.Printf("Restore cache error: %s\n", string(output))
	//	return fmt.Errorf("failed to restore cache from %s: %v", backupFile, err)
	//}

	log.Println("Cache restored successfully from:", backupFile)
	return nil
}
