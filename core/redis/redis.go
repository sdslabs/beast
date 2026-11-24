package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/core/config"
)

var (
	Client    *redis.Client
	redisMu   sync.RWMutex
	KeyPrefix string
)

// Connect initializes the Redis client connection using the provided RedisConfig.
// It verifies that KeyPrefix is non-empty and starts with "beast:", sets up connection options,
// and verifies connectivity with a Ping.
func Connect(cfg config.RedisConfig) (*redis.Client, error) {
	prefix := strings.TrimSpace(cfg.KeyPrefix)
	if prefix == "" {
		return nil, errors.New("redis key prefix cannot be empty")
	}
	if !strings.HasPrefix(prefix, "beast:") {
		return nil, fmt.Errorf("invalid redis key prefix %q: must start with \"beast:\"", prefix)
	}

	opts := &redis.Options{
		Addr:     cfg.Addr(),
		Username: cfg.User,
		Password: cfg.Password.Expose(),
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	}

	if cfg.DialTimeout > 0 {
		opts.DialTimeout = cfg.DialTimeout.Std()
	}
	if cfg.ReadTimeout > 0 {
		opts.ReadTimeout = cfg.ReadTimeout.Std()
	}
	if cfg.WriteTimeout > 0 {
		opts.WriteTimeout = cfg.WriteTimeout.Std()
	}

	if cfg.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := redis.NewClient(opts)

	timeout := cfg.DialTimeout.Std()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to ping redis at %s: %w", cfg.Addr(), err)
	}

	redisMu.Lock()
	Client = rdb
	KeyPrefix = prefix
	redisMu.Unlock()

	return rdb, nil
}

// ConnectWithConfig initializes the Redis client using config.Cfg.RedisConf.
// If Beast is not configured, it returns an error indicating that Beast is not configured.
func ConnectWithConfig() (*redis.Client, error) {
	if !config.IsConfigured() {
		return nil, errors.New("beast is not configured, cannot connect to redis")
	}
	return Connect(config.Cfg.RedisConf)
}

// GetClient returns the active *redis.Client in a thread-safe manner.
func GetClient() *redis.Client {
	redisMu.RLock()
	defer redisMu.RUnlock()
	return Client
}

// FormatKey prefixes the given key with the configured KeyPrefix.
// Beast's Redis ACL user is strictly restricted to ~beast:*, so all keys
// manipulated by the application must begin with the configured prefix.
// If the key or KeyPrefix is empty, it returns an error instead of silently falling back.
func FormatKey(key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", errors.New("redis key cannot be empty")
	}

	redisMu.RLock()
	prefix := KeyPrefix
	redisMu.RUnlock()

	if prefix == "" {
		return "", errors.New("redis key prefix is empty or not configured")
	}
	if strings.HasPrefix(key, prefix) {
		return key, nil
	}
	return prefix + key, nil
}

// Ping checks whether the Redis connection is alive.
func Ping(ctx context.Context) error {
	redisMu.RLock()
	defer redisMu.RUnlock()

	if Client == nil {
		return errors.New("redis client is not initialized")
	}
	return Client.Ping(ctx).Err()
}

// Close gracefully closes the Redis client connection and clears the global references.
func Close() error {
	redisMu.Lock()
	defer redisMu.Unlock()

	KeyPrefix = ""
	if Client != nil {
		err := Client.Close()
		Client = nil
		return err
	}
	return nil
}
