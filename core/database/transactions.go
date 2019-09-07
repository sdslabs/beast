package database

import (
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
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while saving transaction", tx.Error)
	}

	if err := tx.FirstOrCreate(transaction, *transaction).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
