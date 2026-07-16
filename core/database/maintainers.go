package database

import "fmt"

func MigrateChallengeMaintainers() error {
	if Db == nil {
		return fmt.Errorf("database is not initialized")
	}
	return Db.Exec(`
INSERT INTO challenge_maintainers (user_id, challenge_id)
SELECT users.id, challenges.id
FROM users
JOIN challenges ON challenges.author_id = users.id
UNION
SELECT uc.user_id, uc.challenge_id
FROM user_challenges uc
JOIN users ON users.id = uc.user_id
WHERE users.role IN ('author', 'maintainer', 'admin')
ON CONFLICT (user_id, challenge_id) DO NOTHING
`).Error
}

func SetChallengeRelations(challenge *Challenge, tags []*Tag, users []*User) error {
	if challenge == nil || challenge.ID == 0 {
		return fmt.Errorf("persisted challenge is required")
	}
	for _, user := range users {
		if user == nil || user.ID == 0 {
			return fmt.Errorf("persisted challenge manager is required")
		}
	}

	DBMux.Lock()
	defer DBMux.Unlock()
	tx := Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Model(challenge).Association("Tags").Replace(tags); err != nil {
		tx.Rollback()
		return fmt.Errorf("set challenge tags: %w", err)
	}
	if err := tx.Model(challenge).Association("Users").Replace(users); err != nil {
		tx.Rollback()
		return fmt.Errorf("set challenge managers: %w", err)
	}
	return tx.Commit().Error
}
