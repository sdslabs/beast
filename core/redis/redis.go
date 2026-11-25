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

func ConnectWithConfig() (*redis.Client, error) {
	if !config.IsConfigured() {
		return nil, errors.New("beast is not configured, cannot connect to redis")
	}
	return Connect(config.Cfg.RedisConf)
}

func GetClient() *redis.Client {
	redisMu.RLock()
	defer redisMu.RUnlock()
	return Client
}

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
func FormatKeys(keys ...string) ([]string, error) {
	formatted := make([]string, len(keys))
	for i, k := range keys {
		fk, err := FormatKey(k)
		if err != nil {
			return nil, err
		}
		formatted[i] = fk
	}
	return formatted, nil
}

func Ping(ctx context.Context) error {
	redisMu.RLock()
	defer redisMu.RUnlock()

	if Client == nil {
		return errors.New("redis client is not initialized")
	}
	return Client.Ping(ctx).Err()
}

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
