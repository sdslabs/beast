package cache

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
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
	User       string
	Password   string
	Host       string
	Port       string
	DB         uint32
	TLS        bool
	CAFile     string
	ServerName string
}

func Configure(user, password, host, port string, db uint32, tlsEnabled bool, caFile, serverName string) {
	cacheConfig = RedisConfig{User: user, Password: password, Host: host, Port: port, DB: db, TLS: tlsEnabled, CAFile: caFile, ServerName: serverName}
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

func NewTLSConfig(enabled bool, caFile, serverName, host string) (*tls.Config, error) {
	if !enabled {
		return nil, nil
	}
	if serverName == "" {
		serverName = host
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName}
	if caFile == "" {
		return tlsConfig, nil
	}
	certificate, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read Redis CA file: %w", err)
	}
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	if !roots.AppendCertsFromPEM(certificate) {
		return nil, fmt.Errorf("Redis CA file contains no certificates")
	}
	tlsConfig.RootCAs = roots
	return tlsConfig, nil
}

// Connect redis
func ConnectCache() error {
	if err := LoadCacheConfig(); err != nil {
		return err
	}
	tlsConfig, err := NewTLSConfig(cacheConfig.TLS, cacheConfig.CAFile, cacheConfig.ServerName, cacheConfig.Host)
	if err != nil {
		return err
	}
	client := redis.NewClient(&redis.Options{
		Addr:         redisAddress(cacheConfig),
		Username:     cacheConfig.User,
		Password:     cacheConfig.Password,
		DB:           int(cacheConfig.DB),
		TLSConfig:    tlsConfig,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.Ping(ctx).Result()
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
		return fmt.Errorf("close cache connection: %w", err)
	}
	Cache = nil
	return nil
}

func BackupAndReset() error {
	if err := LoadCacheConfig(); err != nil {
		return err
	}
	if err := BackupCache(); err != nil {
		return fmt.Errorf("back up cache: %w", err)
	}
	if err := ResetCache(); err != nil {
		return fmt.Errorf("reset cache: %w", err)
	}
	return nil
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

	args := redisCLIConnectionArgs()
	args = append(args, "--rdb", filepath.Join(backupPath, backupFile))

	environment := append(os.Environ(), fmt.Sprintf("REDISCLI_AUTH=%s", cacheConfig.Password))
	output, err := utils.RunCommand(30*time.Minute, environment, "redis-cli", args...)
	if err != nil {
		return fmt.Errorf("back up Redis: %w; output: %s", err, output)
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
	args := append(redisCLIConnectionArgs(), "FLUSHDB")
	environment := append(os.Environ(), fmt.Sprintf("REDISCLI_AUTH=%s", cacheConfig.Password))
	output, err := utils.RunCommand(2*time.Minute, environment, "redis-cli", args...)
	if err != nil {
		return fmt.Errorf("flush Redis database: %w; output: %s", err, output)
	}

	log.Debug("Reset successful.")
	return nil
}

func redisCLIConnectionArgs() []string {
	args := []string{
		"-h", cacheConfig.Host,
		"-p", cacheConfig.Port,
		"-n", strconv.FormatUint(uint64(cacheConfig.DB), 10),
	}
	if cacheConfig.User != "" {
		args = append(args, "--user", cacheConfig.User)
	}
	if cacheConfig.TLS {
		args = append(args, "--tls")
		if cacheConfig.CAFile != "" {
			args = append(args, "--cacert", cacheConfig.CAFile)
		}
		if cacheConfig.ServerName != "" {
			args = append(args, "--sni", cacheConfig.ServerName)
		}
	}
	return args
}

func RestoreCache(backupFile string) error {
	/*
		The primary issue with restoring cache is that it needs to be written to /var/lib and redis needs to be restarted.
		Redis will then pick up the changes and continue from there.
	*/
	return fmt.Errorf("Redis restore is not implemented; no data was changed")
}
