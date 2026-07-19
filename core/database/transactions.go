package database

import (
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Transaction struct {
	gorm.Model

	Action      string
	UserID      uint `gorm:"not null"`
	ChallengeID uint `gorm:"not null"`
}

func SaveTransaction(transaction *Transaction) error {
	if transaction == nil || transaction.UserID == 0 || transaction.ChallengeID == 0 || transaction.Action == "" {
		return errors.New("complete transaction audit record is required")
	}
	DBMux.Lock()
	defer DBMux.Unlock()

	if err := Db.Create(transaction).Error; err != nil {
		return fmt.Errorf("save transaction audit record: %w", err)
	}
	return nil
}
