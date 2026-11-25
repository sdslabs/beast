package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateDeploymentEndpoint(ctx context.Context, db *gorm.DB, endpoint *models.DeploymentEndpoint) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if endpoint.ID == uuid.Nil {
		endpoint.ID = uuid.New()
	}
	if err := db.Create(endpoint).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func ListEndpointsByDeploymentID(ctx context.Context, db *gorm.DB, deploymentID uuid.UUID) ([]models.DeploymentEndpoint, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var endpoints []models.DeploymentEndpoint
	if err := db.Where("deployment_id = ?", deploymentID).Order("port ASC").Find(&endpoints).Error; err != nil {
		return nil, MapError(err)
	}
	return endpoints, nil
}

func ListActiveEndpointPortsByNode(ctx context.Context, db *gorm.DB, nodeID string) ([]int, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var ports []int
	err := db.Table("deployment_endpoints").
		Distinct("deployment_endpoints.port").
		Joins("INNER JOIN challenge_deployments ON challenge_deployments.id = deployment_endpoints.deployment_id").
		Where("challenge_deployments.node_id = ? AND challenge_deployments.cleaned_up_at IS NULL", nodeID).
		Order("deployment_endpoints.port ASC").
		Pluck("deployment_endpoints.port", &ports).Error
	if err != nil {
		return nil, MapError(err)
	}
	return ports, nil
}

func DeleteEndpointsByDeploymentID(ctx context.Context, db *gorm.DB, deploymentID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if err := db.Where("deployment_id = ?", deploymentID).Delete(&models.DeploymentEndpoint{}).Error; err != nil {
		return MapError(err)
	}
	return nil
}
