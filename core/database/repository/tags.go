package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateTag(ctx context.Context, db *gorm.DB, tag *models.Tag) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if tag.ID == uuid.Nil {
		tag.ID = uuid.New()
	}
	if err := db.Create(tag).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetTagByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.Tag, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var tag models.Tag
	if err := db.Where("id = ?", id).First(&tag).Error; err != nil {
		return nil, MapError(err)
	}
	return &tag, nil
}

func GetTagByName(ctx context.Context, db *gorm.DB, name string) (*models.Tag, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var tag models.Tag
	if err := db.Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, MapError(err)
	}
	return &tag, nil
}

func ListTags(ctx context.Context, db *gorm.DB) ([]models.Tag, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var tags []models.Tag
	if err := db.Order("name ASC").Find(&tags).Error; err != nil {
		return nil, MapError(err)
	}
	return tags, nil
}

func UpdateTag(ctx context.Context, db *gorm.DB, tag *models.Tag) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.Tag{}).Where("id = ?", tag.ID).Update("name", tag.Name)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func DeleteTag(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Where("id = ?", id).Delete(&models.Tag{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func AddChallengeTag(ctx context.Context, db *gorm.DB, challengeID uuid.UUID, tagID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	ct := models.ChallengeTag{
		ChallengeID: challengeID,
		TagID:       tagID,
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&ct).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func RemoveChallengeTag(ctx context.Context, db *gorm.DB, challengeID uuid.UUID, tagID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Where("challenge_id = ? AND tag_id = ?", challengeID, tagID).
		Delete(&models.ChallengeTag{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func ListTagsByChallengeID(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) ([]models.Tag, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var tags []models.Tag
	err := db.Table("tags").
		Joins("INNER JOIN challenge_tags ON challenge_tags.tag_id = tags.id").
		Where("challenge_tags.challenge_id = ?", challengeID).
		Order("tags.name ASC").
		Find(&tags).Error
	if err != nil {
		return nil, MapError(err)
	}
	return tags, nil
}

func ListChallengeTagsByChallengeIDs(ctx context.Context, db *gorm.DB, challengeIDs []uuid.UUID) (map[uuid.UUID][]models.Tag, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := make(map[uuid.UUID][]models.Tag)
	if len(challengeIDs) == 0 {
		return res, nil
	}
	for _, cid := range challengeIDs {
		res[cid] = []models.Tag{}
	}

	type tagJoinRow struct {
		ChallengeID uuid.UUID `gorm:"column:challenge_id"`
		ID          uuid.UUID `gorm:"column:id"`
		Name        string    `gorm:"column:name"`
	}

	var rows []tagJoinRow
	err := db.Table("tags").
		Select("challenge_tags.challenge_id, tags.id, tags.name").
		Joins("INNER JOIN challenge_tags ON challenge_tags.tag_id = tags.id").
		Where("challenge_tags.challenge_id IN ?", challengeIDs).
		Order("tags.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, MapError(err)
	}

	for _, r := range rows {
		res[r.ChallengeID] = append(res[r.ChallengeID], models.Tag{
			ID:   r.ID,
			Name: r.Name,
		})
	}
	return res, nil
}

func ListChallengesByTagID(ctx context.Context, db *gorm.DB, tagID uuid.UUID) ([]models.Challenge, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var challenges []models.Challenge
	err := db.Table("challenges").
		Joins("INNER JOIN challenge_tags ON challenge_tags.challenge_id = challenges.id").
		Where("challenge_tags.tag_id = ?", tagID).
		Order("challenges.created_at ASC").
		Find(&challenges).Error
	if err != nil {
		return nil, MapError(err)
	}
	return challenges, nil
}

func ListChallengesByTagName(ctx context.Context, db *gorm.DB, tagName string) ([]models.Challenge, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var challenges []models.Challenge
	err := db.Table("challenges").
		Joins("INNER JOIN challenge_tags ON challenge_tags.challenge_id = challenges.id").
		Joins("INNER JOIN tags ON tags.id = challenge_tags.tag_id").
		Where("tags.name = ?", tagName).
		Order("challenges.created_at ASC").
		Find(&challenges).Error
	if err != nil {
		return nil, MapError(err)
	}
	return challenges, nil
}
