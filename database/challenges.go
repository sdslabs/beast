package database

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// The `challenges` table has the following columns
// challenge_id
// name
// author
// format
// container_id
// image_id
// status
type Challenge struct {
	gorm.Model

	ChallengeId string `gorm:"not null;type:varchar(64);unique"`
	Name        string `gorm:"not null;type:varchar(64);unique"`
	Author      string `gorm:"not null"`
	Format      string `gorm:"not null"`
	ContainerId string `gorm:"size:64;unique"`
	ImageId     string `gorm:"size:64;unique"`
	Status      string `gorm:"not null;default:'Unknown'"`
	Ports       []Port
}

// Create an entry for the challenge in the Challenge table
// It returns an error if anything wrong happen during the
// transaction.
func CreateChallengeEntry(challenge *Challenge) error {
	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while starting transaction", tx.Error)
	}

	if err := tx.FirstOrCreate(challenge).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Query challenges table to get all the entries in the table
func QueryAllChallenges() ([]Challenge, error) {
	var challenges []Challenge

	tx := Db.Find(&challenges)
	if tx.RecordNotFound() {
		return nil, nil
	}

	return challenges, tx.Error
}

// Queries all the challenges entries where the column represented by key
// have the value in value.
func QueryChallengeEntries(key string, value string) ([]Challenge, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var challenges []Challenge
	tx := Db.Where(queryKey, value).Find(&challenges)
	if tx.RecordNotFound() {
		return nil, nil
	}

	if tx.Error != nil {
		return challenges, tx.Error
	}

	return challenges, nil
}

// Using the column value in key and value in value get the first
// result of the query.
func QueryFirstChallengeEntry(key string, value string) (Challenge, error) {
	challenges, err := QueryChallengeEntries(key, value)
	if err != nil {
		return Challenge{}, err
	}

	if len(challenges) == 0 {
		return Challenge{}, nil
	}

	return challenges[0], nil
}

// This function updates a challenge entry in the database, whereMap is the map
// which contains key value pairs of column and values to filter out the record
// to update. chall is the Challenge variable with the values to update with.
// This function returns any error that might occur while updating the challenge
// entry which includes the error in case the challenge already does not exist in the
// database.
func UpdateChallengeEntry(whereMap map[string]interface{}, chall Challenge) error {
	var challenge Challenge

	tx := Db.Where(whereMap).First(&challenge)
	if tx.RecordNotFound() {
		return fmt.Errorf("No challenge entry to update : WhereClause : %s", whereMap)
	}

	if tx.Error != nil {
		return tx.Error
	}

	// Update the found entry
	tx = Db.Model(&challenge).Updates(chall)

	return tx.Error
}
