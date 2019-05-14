package database

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type UserDetail struct {
	gorm.Model

	UserName   string `gorm:"not null";unique`
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

func UpdateUser(userDetail *UserDetail, chall *Challenge) error {

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while accessing database", tx.Error)
	}

	if err := tx.Model(userDetail).Update("TotalScore", userDetail.TotalScore+chall.Score).Error; err != nil {
		return fmt.Errorf("Error while updating score", tx.Error)
	}

	return tx.Commit().Error
}
