package database

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func MigrateChallengeIdentifiers() error {
	if Db == nil {
		return errors.New("database is not initialized")
	}
	return Db.Transaction(func(tx *gorm.DB) error {
		statements := []string{
			`ALTER TABLE challenges DROP CONSTRAINT IF EXISTS uni_challenges_container_id`,
			`ALTER TABLE challenges DROP CONSTRAINT IF EXISTS challenges_container_id_key`,
			`ALTER TABLE challenges DROP CONSTRAINT IF EXISTS uni_challenges_image_id`,
			`ALTER TABLE challenges DROP CONSTRAINT IF EXISTS challenges_image_id_key`,
			`DROP INDEX IF EXISTS idx_challenges_container_id`,
			`DROP INDEX IF EXISTS uix_challenges_container_id`,
			`DROP INDEX IF EXISTS idx_challenges_image_id`,
			`DROP INDEX IF EXISTS uix_challenges_image_id`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_challenges_active_container_id ON challenges (container_id) WHERE container_id <> '' AND deleted_at IS NULL`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_challenges_active_image_id ON challenges (image_id) WHERE image_id <> '' AND deleted_at IS NULL`,
		}
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("migrate challenge identifiers: %w", err)
			}
		}
		return nil
	})
}
