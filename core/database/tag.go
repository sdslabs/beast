package database

import (
	"errors"
	"fmt"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"gorm.io/gorm"
)

type Tag struct {
	gorm.Model

	Challenges []*Challenge `gorm:"many2many:tag_challenges;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
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

	DBMux.RLock()
	defer DBMux.RUnlock()

	if err := Db.Where(&Tag{TagName: tag.TagName}).First(&tagName).Error; err != nil {
		return nil, err
	}

	if err := Db.Preload("Tags").Preload("Ports").Model(&tagName).Association("Challenges").Find(&challenges); err != nil {
		return challenges, err
	}

	return challenges, nil
}

// Query Related Challenges Metadata
func QueryRelatedChallengesMetadata(tag *Tag) ([]Challenge, error) {
	var challenges []Challenge
	var tagName Tag

	DBMux.RLock()
	defer DBMux.RUnlock()

	if err := Db.Where(&Tag{TagName: tag.TagName}).First(&tagName).Error; err != nil {
		return nil, err
	}

	if err := Db.Model(&tagName).
		Select("id", "name", "created_at", "points", "difficulty", "instanced", "instance_expiration", "status").
		Preload("Tags").
		Association("Challenges").
		Find(&challenges); err != nil {
		return challenges, err
	}

	return challenges, nil
}

// Query using map
func QueryTags(whereMap map[string]interface{}) ([]*Tag, error) {
	var tags []*Tag

	DBMux.RLock()
	defer DBMux.RUnlock()

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
func UpdateTags(tagEntries []*Tag, chall *Challenge) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Transaction(func(tx *gorm.DB) error {
		for _, tagEntry := range tagEntries {
			if tagEntry == nil || tagEntry.TagName == "" {
				return errors.New("tag name is required")
			}
			if err := tx.FirstOrCreate(tagEntry, Tag{TagName: tagEntry.TagName}).Error; err != nil {
				return err
			}
		}
		return tx.Model(chall).Association("Tags").Replace(tagEntries)
	})
}
