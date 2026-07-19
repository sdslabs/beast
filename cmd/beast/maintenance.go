package main

import (
	"fmt"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
)

func initializeMaintenance(useCache bool) (func(), error) {
	if err := config.InitConfig(); err != nil {
		return nil, err
	}
	lock, err := acquireControllerLock(core.BEAST_GLOBAL_DIR)
	if err != nil {
		return nil, fmt.Errorf("maintenance requires an exclusive controller lock: %w", err)
	}
	if useCache {
		redis := config.Cfg.RedisConf
		cache.Configure(redis.User, redis.Password, redis.Host, redis.Port, redis.Db, redis.TLS, redis.CAFile, redis.ServerName)
	}
	return func() { releaseControllerLock(lock) }, nil
}

func requireDestructiveConfirmation() error {
	if !ConfirmDestructive {
		return fmt.Errorf("destructive operation requires --yes")
	}
	return nil
}
