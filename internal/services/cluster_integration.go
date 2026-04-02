package services

import (
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/ketches/ketches/pkg/uuid"
)

func ListClusterIntegrations(clusterID string) ([]entities.ClusterIntegration, error) {
	var integrations []entities.ClusterIntegration
	if err := db.DB.Where("cluster_id = ?", clusterID).Order("created_at").Find(&integrations).Error; err != nil {
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
	if err := db.DB.Where("cluster_id = ? AND integration_type = ? AND enabled = ?", clusterID, integrationType, true).First(&integration).Error; err != nil {
		return nil, err
	}
	return &integration, nil
}

func HasPrometheusIntegration(clusterID string) (bool, error) {
	if db.DB == nil {
		return false, nil
	}

	var count int64
	if err := db.DB.Model(&entities.ClusterIntegration{}).
		Where("cluster_id = ? AND integration_type = ? AND enabled = ?", clusterID, entities.IntegrationTypePrometheus, true).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func CreateClusterIntegration(clusterID string, req *models.CreateClusterIntegrationRequest) (*entities.ClusterIntegration, error) {
	password, err := secrets.EncryptString(req.Password)
	if err != nil {
		return nil, err
	}
	token, err := secrets.EncryptString(req.Token)
	if err != nil {
		return nil, err
	}
	caCert, err := secrets.EncryptString(req.CACert)
	if err != nil {
		return nil, err
	}

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
		Password:        password,
		Token:           token,
		CACert:          caCert,
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
	if req.ClearPassword != nil && *req.ClearPassword {
		integration.Password = ""
	}
	if req.Password != nil {
		password, err := secrets.EncryptString(*req.Password)
		if err != nil {
			return nil, err
		}
		integration.Password = password
	}
	if req.ClearToken != nil && *req.ClearToken {
		integration.Token = ""
	}
	if req.Token != nil {
		token, err := secrets.EncryptString(*req.Token)
		if err != nil {
			return nil, err
		}
		integration.Token = token
	}
	if req.ClearCACert != nil && *req.ClearCACert {
		integration.CACert = ""
	}
	if req.CACert != nil {
		caCert, err := secrets.EncryptString(*req.CACert)
		if err != nil {
			return nil, err
		}
		integration.CACert = caCert
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
