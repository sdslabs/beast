package database

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type Hint struct {
	HintID      uint   `gorm:"primaryKey;autoIncrement"`
	ChallengeID uint   `gorm:"not null"`
	Points      uint   `gorm:"not null"`
	Description string `gorm:"size:255"`
}

type UserHint struct {
	UserID      uint      `gorm:"not null"`
	ChallengeID uint      `gorm:"not null"`
	HintID      uint      `gorm:"not null"`
	Hint        Hint      `gorm:"foreignKey:HintID;references:HintID"`
	Challenge   Challenge `gorm:"foreignKey:ChallengeID;references:ID"`
}

func CreateHintEntry(hint *Hint) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("error while starting transaction: %w", tx.Error)
	}

	if err := tx.FirstOrCreate(hint, *hint).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func GetHintByID(hintID uint) (*Hint, error) {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("error while starting transaction: %w", tx.Error)
	}

	var hint Hint
	if err := tx.Where("hint_id = ?", hintID).First(&hint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return nil, fmt.Errorf("not_found")
		}
		tx.Rollback()
		return nil, fmt.Errorf("db_error")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("commit error: %w", err)
	}

	return &hint, nil
}

// checks if user has already taken the hint
func UserHasTakenHint(userID, hintID uint) (bool, error) {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()
	if tx.Error != nil {
		return false, fmt.Errorf("error while starting transaction: %w", tx.Error)
	}

	var userHint UserHint
	if err := tx.Where("user_id = ? AND hint_id = ?", userID, hintID).First(&userHint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return false, nil
		}
		tx.Rollback()
		return false, fmt.Errorf("db_error")
	}

	if err := tx.Commit().Error; err != nil {
		return false, fmt.Errorf("commit error: %w", err)
	}

	return true, nil
}

func SaveUserHint(userID, challengeID, hintID uint) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("error while starting transaction: %w", tx.Error)
	}

	// Get the hint to check its points
	var hint Hint
	if err := tx.First(&hint, "hint_id = ?", hintID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Get the user to update their points
	var user User
	if err := tx.First(&user, "id = ?", userID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Check if user has enough points
	if user.Score < hint.Points {
		tx.Rollback()
		return fmt.Errorf("not enough points to take this hint")
	}

	// Deduct points from user
	user.Score -= hint.Points

	// Update user's score
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Save the hint usage
	userHint := UserHint{
		UserID:      userID,
		ChallengeID: challengeID,
		HintID:      hintID,
	}

	if err := tx.Create(&userHint).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// QueryHintsTaken retrieves the hints taken by a user for a specific challenge.
func QueryHintsTaken(userID, challengeID uint) ([]Hint, error) {
	var userHints []UserHint
	var hints []Hint

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("user_id = ? AND challenge_id = ?", userID, challengeID).Find(&userHints)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	for _, userHint := range userHints {
		var hint Hint
		tx := Db.First(&hint, userHint.HintID)
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			continue
		}

		if tx.Error != nil {
			return nil, tx.Error
		}

		hints = append(hints, hint)
	}

	return hints, nil
}

func QueryHintsByChallengeID(challengeID uint) ([]Hint, error) {
	var hints []Hint

	DBMux.Lock()
	defer DBMux.Unlock()

	if err := Db.Where("challenge_id = ?", challengeID).Find(&hints).Error; err != nil {
		return nil, err
	}

	return hints, nil
}
