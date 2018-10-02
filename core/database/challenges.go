package database

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Challenge struct {
	gorm.Model

	ChallengeId string
	Name        string
	Author      string
	Format      string
	ContainerId string `gorm:"size:64"`
	ImageId     string `gorm:"size:64"`
	Ports       string
	Status      string
}

// Create an entry for the challenge in the Challenge table
// It returns an error if anything wrong happen during the
// transaction.
func CreateChallengeEntry(challenge Challenge) error {
	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while starting transaction", tx.Error)
	}

	if err := tx.Create(challenge).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Query challenges table to get all the entries in the table
func QueryAllChallenges() ([]Challenge, error) {
	var challenges []Challenge

	err := Db.Find(&challenges).Error
	return challenges, err
}

// Queries all the challenges entries where the column represented by key
// have the value in value.
func QueryChallengeEntries(key string, value string) ([]Challenge, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var challenges []Challenge
	err := Db.Where(queryKey, value).Find(&challenges).Error
	if err != nil {
		return challenges, err
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

	return challenges[0], nil
}
