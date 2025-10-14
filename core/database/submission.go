package database

import (
	"fmt"
	"time"

)

type Submission struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UserID      uint
	ChallengeID uint
	Flag        string `gorm:"type:text"`
	Success     bool
}

func CreateSubmissionEntry(submission *Submission) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("error while starting transaction: %s", tx.Error)
	}

	if err := tx.Create(submission).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func GetAllSubmissions() ([]Submission, error) {
	var submissions []Submission

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Find(&submissions)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return submissions, nil
}

func GetSubmissionsByUser(userID uint) ([]Submission, error) {
	var submissions []Submission

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("user_id = ?", userID).Find(&submissions)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return submissions, nil
}

func GetAllCorrectSubmissions() ([]Submission, error) {
	var submissions []Submission

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("success = ?", true).Find(&submissions)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return submissions, nil
}

func GetSubmissionsByChallenge(challengeID uint) ([]Submission, error) {
	var submissions []Submission

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("challenge_id = ?", challengeID).Find(&submissions)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return submissions, nil
}