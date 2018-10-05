package database

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Author struct {
	gorm.Model

	Challenges []Challenge
	Name       string `gorm:"not null"`
	SshKey     string
	Email      string `gorm:"non null"`
}

// Queries all the authors entries where the column represented by key
// have the value in value.
func QueryAuthorEntries(key string, value string) ([]Author, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var authors []Author
	tx := Db.Where(queryKey, value).Find(&authors)
	if tx.RecordNotFound() {
		return nil, nil
	}

	if tx.Error != nil {
		return authors, tx.Error
	}

	return authors, nil
}

// Using the column value in key and value in value get the first
// result of the query.
func QueryFirstAuthorEntry(key string, value string) (Author, error) {
	authors, err := QueryAuthorEntries(key, value)
	if err != nil {
		return Author{}, err
	}

	if len(authors) == 0 {
		return Author{}, nil
	}

	return authors[0], nil
}

// Create an entry for the author in the Author table
// It returns an error if anything wrong happen during the
// transaction.
func CreateAuthorEntry(author *Author) error {
	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while starting transaction", tx.Error)
	}

	if err := tx.FirstOrCreate(author).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
