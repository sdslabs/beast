package repository

import "errors"

var (
	ErrNoRedis      = errors.New("redis client is not initialized")
	ErrNotFound     = errors.New("record not found in redis")
	ErrInvalidLimit = errors.New("limit must be greater than 0")
)
