package database

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Tag struct {
	gorm.Model

	Challenges []*Challenge `gorm:"many2many:tag_challenges;"`
	TagName    string       `gorm:"not null;unique"`
}

// Queries or Create if not Exist
func QueryOrCreateTagEntry(tag *Tag) error {
	DBMux.Lock()
	defer DBMux.Unlock()
	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while starting transaction", tx.Error)
	}

	if err := tx.FirstOrCreate(tag, *tag).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Query Related Challenges
func QueryRelatedChallenges(tag *Tag) ([]*Challenge, error) {
	var challenges []*Challenge

	DBMux.Lock()
	defer DBMux.Unlock()

	if err := Db.Model(tag).Related(&challenges, "Challenges").Error; err != nil {
		return challenges, err
	}

	return challenges, nil
}

// Query using map
func QueryTags(whereMap map[string]interface{}) ([]*Tag, error) {
	var tags []*Tag

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where(whereMap).Find(&tags)
	if tx.RecordNotFound() {
		return nil, nil
	}

	if tx.Error != nil {
		return tags, tx.Error
	}

	return tags, nil
}
