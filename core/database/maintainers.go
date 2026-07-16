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
