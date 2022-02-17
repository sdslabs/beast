package database

import (
	"errors"
	"fmt"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"gorm.io/gorm"
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
		return fmt.Errorf("error while starting transaction: %s", tx.Error)
	}

	if err := tx.FirstOrCreate(tag, *tag).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Query Related Challenges
func QueryRelatedChallenges(tag *Tag) ([]Challenge, error) {
	var challenges []Challenge
	var tagName Tag

	DBMux.Lock()
	defer DBMux.Unlock()

	Db.Where(&Tag{TagName: tag.TagName}).First(&tagName)

	if err := Db.Preload("Tags").Preload("Ports").Model(&tagName).Association("Challenges").Find(&challenges); err != nil {
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
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return tags, tx.Error
	}

	return tags, nil
}

// Update tags
func UpdateTags(tag []*Tag, chall *Challenge) error {
	var tags []Tag

	DBMux.Lock()
	defer DBMux.Unlock()

	// Delete existing tags
	Db.Model(&chall).Association("Tags").Find(&tags)
	tx := Db.Begin()
	if err := tx.Model(&chall).Association("Tags").Delete(tags); err != nil {
		return err
	}

	// Create tags
	for _, tagEntry := range tag {
		if err := tx.FirstOrCreate(tagEntry, *tagEntry).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Model(&chall).Association("Tags").Append(tagEntry); err != nil {
			return err
		}
	}

	return tx.Commit().Error
}
