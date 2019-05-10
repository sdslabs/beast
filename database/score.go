package database

import (
	"fmt"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Score struct {
	UserID      string `gorm:"not null"`
	ChallengeID uint   `gorm:"not null"`
	Score       int
}

func AddScoreDetails(score Score) error {

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while adding user", tx.Error)
	}

	if err := tx.Create(&score).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
