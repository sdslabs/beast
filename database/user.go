package database

import (
	"fmt"
	"sort"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type UserDetail struct {
	gorm.Model

	UserName   string `gorm:"not null";unique`
	UserEmail  string `gorm:"not null"`
	Password   [32]byte
	TotalScore int
	Challenges []Challenge `gorm:"many2many:Score"`
}

//Adds user info
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

//Update user score
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

// Query user table to get all the entries in the table
func QueryAllUsers() ([]UserDetail, error) {
	var user []UserDetail

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Find(&user)
	if tx.RecordNotFound() {
		return nil, nil
	}

	return user, tx.Error
}

// Query user table by score in decreasing order to get all the entries in the table
func QueryAllUsersByScore() ([]UserDetail, error) {
	var user []UserDetail

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Find(&user)
	if tx.RecordNotFound() {
		return nil, nil
	}
	sort.SliceStable(user, func(i, j int) bool {
		return user[i].TotalScore > user[j].TotalScore
	})
	return user, tx.Error
}

// Query user details by using their informations
func QueryUserEntry(key string, value string) ([]UserDetail, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var userDetail []UserDetail

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where(queryKey, value).Find(&userDetail)
	if tx.RecordNotFound() {
		return nil, nil
	}

	if tx.Error != nil {
		return userDetail, tx.Error
	}

	return userDetail, nil
}
