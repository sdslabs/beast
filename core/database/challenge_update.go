package database

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpdateChallengeConfiguration(challengeID uint, updates map[string]interface{}, ports *[]uint32, tags *[]string) error {
	if challengeID == 0 {
		return fmt.Errorf("persisted challenge is required")
	}

	DBMux.Lock()
	defer DBMux.Unlock()
	return Db.Transaction(func(tx *gorm.DB) error {
		var challenge Challenge
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&challenge, challengeID).Error; err != nil {
			return err
		}
		if len(updates) > 0 {
			if err := tx.Model(&challenge).Omit(clause.Associations).Updates(updates).Error; err != nil {
				return err
			}
		}
		if ports != nil {
			if err := tx.Unscoped().Where("challenge_id = ?", challengeID).Delete(&Port{}).Error; err != nil {
				return err
			}
			for _, portNumber := range *ports {
				if err := tx.Create(&Port{ChallengeID: challengeID, Server: challenge.ServerDeployed, PortNo: portNumber}).Error; err != nil {
					return fmt.Errorf("reserve port %d: %w", portNumber, err)
				}
			}
		}
		if tags != nil {
			tagModels := make([]*Tag, 0, len(*tags))
			for _, tagName := range *tags {
				tag := &Tag{TagName: tagName}
				if err := tx.FirstOrCreate(tag, Tag{TagName: tagName}).Error; err != nil {
					return err
				}
				tagModels = append(tagModels, tag)
			}
			if err := tx.Model(&challenge).Association("Tags").Replace(tagModels); err != nil {
				return err
			}
		}
		return nil
	})
}
