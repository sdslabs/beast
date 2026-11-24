package database

import (
	"context"
	"errors"
	"sync"

	"github.com/sdslabs/beastv4/core/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	Db    *gorm.DB
	dberr error
	DBMux sync.RWMutex
)

// Connect initializes the PostgreSQL connection using the provided config.PsqlConfig.
// It sets the active global Db handle, configures the connection pool, and returns the *gorm.DB instance.
func Connect(cfg config.PsqlConfig) (*gorm.DB, error) {
	return connectDSN(cfg.DSN(), cfg.DbPoolOptions)
}

// connectDSN initializes the PostgreSQL connection using gorm.Open with the provided DSN and pool options.
func connectDSN(dsn string, opts config.DBPoolOptions) (*gorm.DB, error) {
	if dsn == "" {
		dberr = errors.New("empty postgres DSN: host, user, and dbname must be configured")
		return nil, dberr
	}

	if opts.MaxOpenConns <= 0 {
		dberr = errors.New("invalid db pool options: max_open_conns must be greater than 0")
		return nil, dberr
	}
	if opts.MaxIdleConns <= 0 {
		dberr = errors.New("invalid db pool options: max_idle_conns must be greater than 0")
		return nil, dberr
	}
	if opts.ConnMaxLifetime <= 0 {
		dberr = errors.New("invalid db pool options: conn_max_lifetime must be greater than 0")
		return nil, dberr
	}

	DBMux.Lock()
	defer DBMux.Unlock()

	var err error
	Db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		dberr = err
		return nil, dberr
	}

	sqlDB, dbErr := Db.DB()
	if dbErr != nil {
		dberr = dbErr
		return nil, dberr
	}
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)

	dberr = nil
	return Db, nil
}

// GetDb returns the active *gorm.DB instance.
func GetDb() *gorm.DB {
	DBMux.RLock()
	defer DBMux.RUnlock()
	return Db
}

// Ping checks whether the PostgreSQL database connection is alive.
func Ping(ctx context.Context) error {
	DBMux.RLock()
	defer DBMux.RUnlock()

	if Db == nil {
		return errors.New("database connection is not initialized")
	}
	sqlDB, err := Db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the active PostgreSQL connection and clears global handles.
func Close() error {
	DBMux.Lock()
	defer DBMux.Unlock()

	if Db != nil {
		sqlDB, err := Db.DB()
		Db = nil
		dberr = nil
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
