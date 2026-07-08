package database

import (
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DynamicFlagClaim struct {
	gorm.Model

	ChallengeID uint   `gorm:"not null"`
	Flag        string `gorm:"type:text;not null"`
	UserID      uint   `gorm:"not null"`
}

func (DynamicFlagClaim) TableName() string {
	return "dynamic_flag_claims"
}

type DynamicScoreDirty struct {
	gorm.Model

	ChallengeID  uint      `gorm:"not null"`
	LastSolveAt  time.Time `gorm:"not null"`
	LastSolverID uint      `gorm:"not null"`
}

func (DynamicScoreDirty) TableName() string {
	return "dynamic_score_dirty"
}

type SubmissionAttemptStatus uint8

const (
	SubmissionAttemptAccepted SubmissionAttemptStatus = iota
	SubmissionAttemptAlreadySolved
	SubmissionAttemptMaxAttempts
)

type SubmissionAttemptResult struct {
	Status SubmissionAttemptStatus
	Tries  uint
}

type SolveFinalizationResult struct {
	Awarded      bool
	Points       uint
	DynamicDelta int64
}

type DynamicFlagClaimStatus uint8

const (
	DynamicFlagClaimCreated DynamicFlagClaimStatus = iota
	DynamicFlagClaimedBySameUser
	DynamicFlagClaimedByOtherUser
)

type DynamicFlagClaimResult struct {
	Status      DynamicFlagClaimStatus
	ClaimedByID uint
}

func MigrateSubmissionGuards() error {
	if err := dedupeUserChallengeRows(); err != nil {
		return err
	}

	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_challenges_user_challenge ON user_challenges (user_id, challenge_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_dynamic_flag_claims_challenge_flag ON dynamic_flag_claims (challenge_id, flag)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_dynamic_score_dirty_challenge ON dynamic_score_dirty (challenge_id)`,
	}

	for _, statement := range statements {
		if err := Db.Exec(statement).Error; err != nil {
			return err
		}
	}

	return nil
}

func dedupeUserChallengeRows() error {
	return Db.Transaction(func(tx *gorm.DB) error {
		merge := `
WITH ranked AS (
	SELECT
		id,
		user_id,
		challenge_id,
		SUM(tries) OVER (PARTITION BY user_id, challenge_id) AS total_tries,
		BOOL_OR(solved) OVER (PARTITION BY user_id, challenge_id) AS any_solved,
		ROW_NUMBER() OVER (
			PARTITION BY user_id, challenge_id
			ORDER BY solved DESC, created_at ASC, id ASC
		) AS rn
	FROM user_challenges
),
keepers AS (
	SELECT id, total_tries, any_solved
	FROM ranked
	WHERE rn = 1
)
UPDATE user_challenges uc
SET tries = keepers.total_tries,
	solved = keepers.any_solved
FROM keepers
WHERE uc.id = keepers.id`
		if err := tx.Exec(merge).Error; err != nil {
			return fmt.Errorf("failed to merge duplicate user_challenges rows: %w", err)
		}

		removeDuplicates := `
WITH ranked AS (
	SELECT
		id,
		ROW_NUMBER() OVER (
			PARTITION BY user_id, challenge_id
			ORDER BY solved DESC, created_at ASC, id ASC
		) AS rn
	FROM user_challenges
)
DELETE FROM user_challenges uc
USING ranked
WHERE uc.id = ranked.id AND ranked.rn > 1`
		if err := tx.Exec(removeDuplicates).Error; err != nil {
			return fmt.Errorf("failed to delete duplicate user_challenges rows: %w", err)
		}

		return nil
	})
}

func ReserveSubmissionAttempt(userID, challengeID uint, maxAttemptLimit int, flag string, now time.Time) (SubmissionAttemptResult, error) {
	var row struct {
		ID     uint
		Tries  uint
		Solved bool
	}

	tx := Db.Raw(`
INSERT INTO user_challenges (created_at, user_id, challenge_id, tries, solved, flag, cheating)
VALUES (?, ?, ?, 1, false, ?, false)
ON CONFLICT (user_id, challenge_id) DO UPDATE
SET tries = user_challenges.tries + 1,
	flag = EXCLUDED.flag,
	created_at = EXCLUDED.created_at
WHERE user_challenges.solved = false
	AND (? <= 0 OR user_challenges.tries < ?)
RETURNING id, tries, solved`,
		now, userID, challengeID, flag, maxAttemptLimit, maxAttemptLimit,
	).Scan(&row)
	if tx.Error != nil {
		return SubmissionAttemptResult{}, tx.Error
	}

	if tx.RowsAffected > 0 {
		return SubmissionAttemptResult{
			Status: SubmissionAttemptAccepted,
			Tries:  row.Tries,
		}, nil
	}

	var existing UserChallenges
	err := Db.Where("user_id = ? AND challenge_id = ?", userID, challengeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return SubmissionAttemptResult{}, fmt.Errorf("submission attempt was not persisted")
	}
	if err != nil {
		return SubmissionAttemptResult{}, err
	}

	if existing.Solved {
		return SubmissionAttemptResult{
			Status: SubmissionAttemptAlreadySolved,
			Tries:  existing.Tries,
		}, nil
	}

	return SubmissionAttemptResult{
		Status: SubmissionAttemptMaxAttempts,
		Tries:  existing.Tries,
	}, nil
}

func MarkSubmissionSolved(userID, challengeID uint, flag string, cheating bool, now time.Time) (bool, error) {
	tx := Db.Model(&UserChallenges{}).
		Where("user_id = ? AND challenge_id = ? AND solved = ?", userID, challengeID, false).
		Updates(map[string]interface{}{
			"solved":     true,
			"flag":       flag,
			"cheating":   cheating,
			"created_at": now,
		})
	if tx.Error != nil {
		return false, tx.Error
	}

	return tx.RowsAffected > 0, nil
}

func FinalizeSubmissionSolve(userID, challengeID uint, flag string, cheating bool, now time.Time, useDynamicScore bool) (SolveFinalizationResult, error) {
	result := SolveFinalizationResult{}

	err := Db.Transaction(func(tx *gorm.DB) error {
		var challenge Challenge
		query := tx
		if tx.Dialector.Name() == "postgres" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := query.First(&challenge, challengeID).Error; err != nil {
			return err
		}

		result.Points = challenge.Points

		update := tx.Model(&UserChallenges{}).
			Where("user_id = ? AND challenge_id = ? AND solved = ?", userID, challengeID, false).
			Updates(map[string]interface{}{
				"solved":     true,
				"flag":       flag,
				"cheating":   cheating,
				"created_at": now,
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return nil
		}

		result.Awarded = true
		pointsToAward := challenge.Points
		if useDynamicScore {
			solvers, err := countSolvedContestantsForChallengeTx(tx, challengeID)
			if err != nil {
				return err
			}

			pointsToAward = DynamicChallengeScore(challenge.MaxPoints, challenge.MinPoints, solvers)
			result.Points = pointsToAward
			result.DynamicDelta = int64(pointsToAward) - int64(challenge.Points)

			if err := tx.Model(&Challenge{}).Where("id = ?", challengeID).Update("points", pointsToAward).Error; err != nil {
				return err
			}

			if result.DynamicDelta != 0 {
				if err := applyDynamicScoreDeltaToPreviousSolversTx(tx, challengeID, userID, result.DynamicDelta); err != nil {
					return err
				}
			}
		}

		return awardUserScoreTx(tx, userID, int64(pointsToAward))
	})
	if err != nil {
		return SolveFinalizationResult{}, err
	}

	return result, nil
}

func DynamicChallengeScore(maxPoints, minPoints, solvers uint) uint {
	if solvers == 0 || solvers == 1 {
		return maxPoints
	}
	divisor := (1 + math.Pow((float64(solvers)-1)/11.92201, 1.206069))
	return uint(math.Round(float64(minPoints) + (float64(maxPoints)-float64(minPoints))/divisor))
}

func countSolvedContestantsForChallengeTx(tx *gorm.DB, challengeID uint) (uint, error) {
	var count int64
	err := tx.Table("user_challenges").
		Joins("JOIN users ON users.id = user_challenges.user_id").
		Where("user_challenges.challenge_id = ? AND user_challenges.solved = ? AND users.role = ?", challengeID, true, "contestant").
		Count(&count).Error
	return uint(count), err
}

func applyDynamicScoreDeltaToPreviousSolversTx(tx *gorm.DB, challengeID, newSolverID uint, delta int64) error {
	return tx.Exec(`
UPDATE users
SET score = CASE WHEN users.score + ? < 0 THEN 0 ELSE users.score + ? END
FROM user_challenges
WHERE users.id = user_challenges.user_id
	AND user_challenges.challenge_id = ?
	AND user_challenges.solved = true
	AND users.role = ?
	AND users.id <> ?`,
		delta, delta, challengeID, "contestant", newSolverID,
	).Error
}

func awardUserScoreTx(tx *gorm.DB, userID uint, delta int64) error {
	return tx.Model(&User{}).
		Where("id = ?", userID).
		UpdateColumn("score", gorm.Expr("CASE WHEN score + ? < 0 THEN 0 ELSE score + ? END", delta, delta)).
		Error
}

func MarkSubmissionCheating(userID, challengeID uint, flag string) error {
	return Db.Model(&UserChallenges{}).
		Where("user_id = ? AND challenge_id = ?", userID, challengeID).
		Updates(map[string]interface{}{
			"flag":     flag,
			"cheating": true,
		}).Error
}

func ClaimDynamicFlag(challengeID, userID uint, flag string, now time.Time) (DynamicFlagClaimResult, error) {
	var row struct {
		UserID uint
	}

	tx := Db.Raw(`
INSERT INTO dynamic_flag_claims (created_at, updated_at, challenge_id, flag, user_id)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (challenge_id, flag) DO NOTHING
RETURNING user_id`,
		now, now, challengeID, flag, userID,
	).Scan(&row)
	if tx.Error != nil {
		return DynamicFlagClaimResult{}, tx.Error
	}

	if tx.RowsAffected > 0 {
		return DynamicFlagClaimResult{
			Status:      DynamicFlagClaimCreated,
			ClaimedByID: userID,
		}, nil
	}

	var claim DynamicFlagClaim
	err := Db.Where("challenge_id = ? AND flag = ?", challengeID, flag).First(&claim).Error
	if err != nil {
		return DynamicFlagClaimResult{}, err
	}

	if claim.UserID == userID {
		return DynamicFlagClaimResult{
			Status:      DynamicFlagClaimedBySameUser,
			ClaimedByID: claim.UserID,
		}, nil
	}

	return DynamicFlagClaimResult{
		Status:      DynamicFlagClaimedByOtherUser,
		ClaimedByID: claim.UserID,
	}, nil
}

func AwardUserScore(userID uint, delta int64) error {
	return Db.Model(&User{}).
		Where("id = ?", userID).
		UpdateColumn("score", gorm.Expr("CASE WHEN score + ? < 0 THEN 0 ELSE score + ? END", delta, delta)).
		Error
}

func MarkDynamicScoreDirty(challengeID, userID uint, solvedAt time.Time) error {
	return Db.Exec(`
INSERT INTO dynamic_score_dirty (created_at, updated_at, challenge_id, last_solve_at, last_solver_id)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (challenge_id) DO UPDATE
SET updated_at = EXCLUDED.updated_at,
	last_solve_at = EXCLUDED.last_solve_at,
	last_solver_id = EXCLUDED.last_solver_id`,
		solvedAt, solvedAt, challengeID, solvedAt, userID,
	).Error
}

func QueryDirtyDynamicScores(limit int) ([]DynamicScoreDirty, error) {
	var dirty []DynamicScoreDirty
	err := Db.Order("updated_at ASC").Limit(limit).Find(&dirty).Error
	return dirty, err
}

func ClearDynamicScoreDirty(challengeID uint, seenUpdatedAt time.Time) error {
	return Db.Unscoped().
		Where("challenge_id = ? AND updated_at <= ?", challengeID, seenUpdatedAt).
		Delete(&DynamicScoreDirty{}).
		Error
}

func CountSolvedSubmissionsForChallenge(challengeID uint) (uint, error) {
	var count int64
	err := Db.Table("user_challenges").
		Joins("JOIN users ON users.id = user_challenges.user_id").
		Where("user_challenges.challenge_id = ? AND user_challenges.solved = ? AND users.role = ?", challengeID, true, "contestant").
		Count(&count).Error
	return uint(count), err
}

func ApplyDynamicScoreDelta(challengeID uint, newPoints uint, delta int64) error {
	return Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Challenge{}).Where("id = ?", challengeID).Update("points", newPoints).Error; err != nil {
			return err
		}

		if delta == 0 {
			return nil
		}

		return tx.Exec(`
UPDATE users
SET score = CASE WHEN users.score + ? < 0 THEN 0 ELSE users.score + ? END
FROM user_challenges
WHERE users.id = user_challenges.user_id
	AND user_challenges.challenge_id = ?
	AND user_challenges.solved = true
	AND users.role = ?`,
			delta, delta, challengeID, "contestant",
		).Error
	})
}
