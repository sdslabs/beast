package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrNotFound            = errors.New("record not found")
	ErrAlreadyExists       = errors.New("record already exists")
	ErrForeignKeyViolation = errors.New("foreign key constraint violation")
	ErrCheckViolation      = errors.New("check constraint violation")
	ErrNotNullViolation    = errors.New("not null constraint violation")
	ErrDatabase            = errors.New("database error")
	ErrNoDatabase          = errors.New("database connection is not initialized")
	ErrInvalidLimit        = errors.New("limit must be greater than 0")
)

func MapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAlreadyExists
	}

	if errors.Is(err, gorm.ErrForeignKeyViolated) {
		return ErrForeignKeyViolation
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return ErrAlreadyExists
		case "23503": // foreign_key_violation
			return ErrForeignKeyViolation
		case "23502": // not_null_violation
			return ErrNotNullViolation
		case "23514": // check_violation
			return ErrCheckViolation
		}
	}

	return err
}
