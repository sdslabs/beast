package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateDeployment(deployment *models.ChallengeDeployment) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if deployment.ID == uuid.Nil {
		deployment.ID = uuid.New()
	}
	if deployment.CreatedAt.IsZero() {
		deployment.CreatedAt = time.Now()
	}
	if deployment.NodeID == "" {
		deployment.NodeID = "local"
	}
	if deployment.Status == "" {
		deployment.Status = models.DeploymentStatusStarting
	}
	if err := db.Create(deployment).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetDeploymentByID(id uuid.UUID) (*models.ChallengeDeployment, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var d models.ChallengeDeployment
	if err := db.Where("id = ?", id).First(&d).Error; err != nil {
		return nil, MapError(err)
	}
	return &d, nil
}

func GetActiveDeployment(challengeID, teamID uuid.UUID) (*models.ChallengeDeployment, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var d models.ChallengeDeployment
	q := db.Where("challenge_id = ? AND cleaned_up_at IS NULL", challengeID)
	if teamID != uuid.Nil {
		q = q.Where("owner_user_id IS NULL OR owner_user_id IN (SELECT id FROM users WHERE team_id = ?)", teamID)
	} else {
		q = q.Where("owner_user_id IS NULL")
	}

	if err := q.Order("owner_user_id DESC NULLS LAST").First(&d).Error; err != nil {
		return nil, MapError(err)
	}
	return &d, nil
}

func ListDeployments(offset, limit int) ([]models.ChallengeDeployment, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var list []models.ChallengeDeployment
	q := db.Model(&models.ChallengeDeployment{}).Order("created_at DESC").Limit(limit).Offset(offset)
	if err := q.Find(&list).Error; err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func ListDeploymentsByNodeID(nodeID string) ([]models.ChallengeDeployment, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var list []models.ChallengeDeployment
	if err := db.Where("node_id = ?", nodeID).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func ListExpiredDeployments(now time.Time) ([]models.ChallengeDeployment, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var list []models.ChallengeDeployment
	err := db.Where("owner_user_id IS NOT NULL AND expires_at IS NOT NULL AND expires_at <= ? AND status NOT IN (?, ?) AND cleaned_up_at IS NULL",
		now, models.DeploymentStatusStopped, models.DeploymentStatusExpired).
		Find(&list).Error
	if err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func ListDeploymentsNeedingCleanup() ([]models.ChallengeDeployment, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var list []models.ChallengeDeployment
	err := db.Where("cleaned_up_at IS NULL AND status IN (?, ?, ?)",
		models.DeploymentStatusStopped, models.DeploymentStatusExpired, models.DeploymentStatusError).
		Find(&list).Error
	if err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func CountDeploymentsByStatus(status models.DeploymentStatus) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.ChallengeDeployment{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountActiveDeploymentsByChallenge(challengeID uuid.UUID) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	err := db.Model(&models.ChallengeDeployment{}).
		Where("challenge_id = ? AND cleaned_up_at IS NULL", challengeID).
		Count(&count).Error
	if err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func UpdateDeploymentStatus(id uuid.UUID, status models.DeploymentStatus) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	updates := map[string]any{"status": status}
	if status == models.DeploymentStatusRunning {
		updates["started_at"] = time.Now()
	}

	res := db.Model(&models.ChallengeDeployment{}).
		Where("id = ?", id).
		Updates(updates)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func UpdateDeploymentRuntimeInfo(id uuid.UUID, runtimeID *string, instanceDir *string) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.ChallengeDeployment{}).Where("id = ?", id).Updates(map[string]any{
		"runtime_identifier": runtimeID,
		"instance_dir":       instanceDir,
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func RenewDeployment(id uuid.UUID, newExpiresAt time.Time) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	now := time.Now()
	res := db.Model(&models.ChallengeDeployment{}).Where("id = ?", id).Updates(map[string]any{
		"renewals_used":   gorm.Expr("renewals_used + 1"),
		"expires_at":      newExpiresAt,
		"last_renewed_at": now,
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func MarkDeploymentStopped(id uuid.UUID, stoppedAt time.Time) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.ChallengeDeployment{}).Where("id = ?", id).Updates(map[string]any{
		"status":     models.DeploymentStatusStopped,
		"stopped_at": stoppedAt,
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func MarkDeploymentCleanedUp(id uuid.UUID, cleanedUpAt time.Time) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.ChallengeDeployment{}).Where("id = ?", id).Update("cleaned_up_at", cleanedUpAt)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
