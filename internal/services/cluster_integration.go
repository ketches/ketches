package services

import (
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

func ListClusterIntegrations(clusterID string) ([]entities.ClusterIntegration, error) {
	var integrations []entities.ClusterIntegration
	if err := db.DB.Where("cluster_id = ?", clusterID).Order("created_at asc").Find(&integrations).Error; err != nil {
		return nil, err
	}
	return integrations, nil
}

func GetClusterIntegration(id string) (*entities.ClusterIntegration, error) {
	var integration entities.ClusterIntegration
	if err := db.DB.First(&integration, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &integration, nil
}

func GetClusterIntegrationByType(clusterID string, integrationType entities.IntegrationType) (*entities.ClusterIntegration, error) {
	var integration entities.ClusterIntegration
	if err := db.DB.Preload("Cluster").Where("cluster_id = ? AND integration_type = ? AND enabled = ?", clusterID, integrationType, true).First(&integration).Error; err != nil {
		return nil, err
	}
	return &integration, nil
}

func CreateClusterIntegration(clusterID string, req *models.CreateClusterIntegrationRequest) (*entities.ClusterIntegration, error) {
	integration := &entities.ClusterIntegration{
		ID:              uuid.New(),
		ClusterID:       clusterID,
		IntegrationType: entities.IntegrationType(req.IntegrationType),
		Name:            req.Name,
		Endpoint:        req.Endpoint,
		Namespace:       req.Namespace,
		ServiceName:     req.ServiceName,
		ServicePort:     req.ServicePort,
		Username:        req.Username,
		Password:        req.Password,
		Token:           req.Token,
		CACert:          req.CACert,
		SkipTLSVerify:   req.SkipTLSVerify,
		Enabled:         req.Enabled,
	}

	if err := db.DB.Create(integration).Error; err != nil {
		return nil, err
	}

	return integration, nil
}

func UpdateClusterIntegration(id string, req *models.UpdateClusterIntegrationRequest) (*entities.ClusterIntegration, error) {
	integration, err := GetClusterIntegration(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		integration.Name = *req.Name
	}
	if req.Endpoint != nil {
		integration.Endpoint = *req.Endpoint
	}
	if req.Namespace != nil {
		integration.Namespace = *req.Namespace
	}
	if req.ServiceName != nil {
		integration.ServiceName = *req.ServiceName
	}
	if req.ServicePort != nil {
		integration.ServicePort = *req.ServicePort
	}
	if req.Username != nil {
		integration.Username = *req.Username
	}
	if req.Password != nil {
		integration.Password = *req.Password
	}
	if req.Token != nil {
		integration.Token = *req.Token
	}
	if req.CACert != nil {
		integration.CACert = *req.CACert
	}
	if req.SkipTLSVerify != nil {
		integration.SkipTLSVerify = *req.SkipTLSVerify
	}
	if req.Enabled != nil {
		integration.Enabled = *req.Enabled
	}

	if err := db.DB.Save(integration).Error; err != nil {
		return nil, err
	}

	return integration, nil
}

func DeleteClusterIntegration(id string) error {
	return db.DB.Delete(&entities.ClusterIntegration{}, "id = ?", id).Error
}
