package database

import (
	"fmt"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type UserDetail struct {
	UserID     string `gorm:"not null"`
	UserEmail  string `gorm:"not null"`
	Password   string
	TotalScore int
	Challenges []Challenge `gorm:"many2many:Score"`
}

func AddUser(userDetail *UserDetail) error {

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while adding user", tx.Error)
	}

	if err := tx.FirstOrCreate(userDetail, *userDetail).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
