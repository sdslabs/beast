package repository

import (
	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateDeploymentEndpoint(endpoint *models.DeploymentEndpoint) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if endpoint.ID == uuid.Nil {
		endpoint.ID = uuid.New()
	}
	if err := db.Create(endpoint).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func ListEndpointsByDeploymentID(deploymentID uuid.UUID) ([]models.DeploymentEndpoint, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var endpoints []models.DeploymentEndpoint
	if err := db.Where("deployment_id = ?", deploymentID).Order("port ASC").Find(&endpoints).Error; err != nil {
		return nil, MapError(err)
	}
	return endpoints, nil
}

func ListActiveEndpointPortsByNode(nodeID string) ([]int, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
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

func DeleteEndpointsByDeploymentID(deploymentID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if err := db.Where("deployment_id = ?", deploymentID).Delete(&models.DeploymentEndpoint{}).Error; err != nil {
		return MapError(err)
	}
	return nil
}
