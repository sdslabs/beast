package main

import (
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	log "github.com/sirupsen/logrus"
)

func initializeCLIRuntime(requireCache, requireRemotes bool) (func(), error) {
	if err := config.InitConfig(); err != nil {
		return nil, err
	}
	auth.Init(core.ITERATIONS, core.HASH_LENGTH, core.TIMEPERIOD, core.ISSUER, config.Cfg.JWTSecret, []string{core.USER_ROLES["author"], core.USER_ROLES["maintainer"]}, []string{core.USER_ROLES["admin"]}, []string{core.USER_ROLES["contestant"]})
	if err := database.Init(); err != nil {
		closeCLIDatabase()
		return nil, err
	}

	cacheInitialized := false
	if requireCache {
		redis := config.Cfg.RedisConf
		cache.Configure(redis.User, redis.Password, redis.Host, redis.Port, redis.Db, redis.TLS, redis.CAFile, redis.ServerName)
		if err := cache.Init(); err != nil {
			closeCLIDatabase()
			return nil, err
		}
		cacheInitialized = true
	}
	if requireRemotes {
		remoteManager.Init()
	}

	return func() {
		if requireRemotes {
			remoteManager.Stop()
		}
		if cacheInitialized {
			if err := cache.Close(); err != nil {
				log.Warnf("close CLI cache: %v", err)
			}
		}
		closeCLIDatabase()
	}, nil
}

func closeCLIDatabase() {
	if database.Db == nil {
		return
	}
	sqlDB, err := database.Db.DB()
	if err != nil {
		log.Warnf("access CLI database connection: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Warnf("close CLI database connection: %v", err)
	}
	database.Db = nil
}
