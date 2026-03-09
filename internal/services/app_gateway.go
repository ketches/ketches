package services

import (
	"context"
	"errors"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

// ListAppGateways returns all gateways for a given app
func ListAppGateways(appID string) ([]models.AppGatewayResponse, error) {
	var gateways []entities.AppGateway
	err := db.DB.Where("app_id = ?", appID).Find(&gateways).Error
	if err != nil {
		return nil, err
	}
	result := make([]models.AppGatewayResponse, 0, len(gateways))
	for _, gw := range gateways {
		result = append(result, toAppGatewayResponse(&gw))
	}
	return result, nil
}

// CreateAppGateway creates a new gateway for an app
func CreateAppGateway(ctx context.Context, appID string, req *models.CreateGatewayRequest) (*models.AppGatewayResponse, error) {
	entity := &entities.AppGateway{
		ID:          uuid.New(),
		AppID:       appID,
		Port:        req.Port,
		Protocol:    req.Protocol,
		Domain:      req.Domain,
		Path:        req.Path,
		GatewayPort: req.GatewayPort,
		Exposed:     req.Exposed,
	}
	if req.CertID != "" {
		certID := req.CertID
		entity.CertID = &certID
	}

	var existing entities.AppGateway
	err := db.DB.Where("app_id = ? AND port = ? AND protocol = ?", appID, req.Port, req.Protocol).First(&existing).Error
	if err == nil {
		return nil, errors.New("gateway with this port and protocol already exists for this app")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create in database
	if err := db.DB.Create(entity).Error; err != nil {
		return nil, err
	}

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	// Sync to Kubernetes cluster
	if err := core.SyncGatewaysToK8s(ctx, appCtx); err != nil {
		return nil, err
	}

	res := toAppGatewayResponse(entity)
	return &res, nil
}

// UpdateAppGateway updates an existing gateway
func UpdateAppGateway(ctx context.Context, id string, req *models.UpdateGatewayRequest) (*models.AppGatewayResponse, error) {
	var gateway entities.AppGateway
	err := db.DB.First(&gateway, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("gateway not found")
		}
		return nil, err
	}

	if req.Port != gateway.Port || req.Protocol != gateway.Protocol {
		var existing entities.AppGateway
		err := db.DB.Where("app_id = ? AND port = ? AND protocol = ? AND id != ?", gateway.AppID, req.Port, req.Protocol, id).First(&existing).Error
		if err == nil {
			return nil, errors.New("gateway with this port and protocol already exists for this app")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Update fields
	gateway.Port = req.Port
	gateway.Protocol = req.Protocol
	gateway.Domain = req.Domain
	gateway.Path = req.Path
	gateway.GatewayPort = req.GatewayPort
	gateway.Exposed = req.Exposed
	if req.CertID != "" {
		certID := req.CertID
		gateway.CertID = &certID
	} else {
		gateway.CertID = nil
	}

	// Save to database
	if err := db.DB.Save(&gateway).Error; err != nil {
		return nil, err
	}

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, gateway.AppID)
	if err != nil {
		return nil, err
	}

	// Sync to Kubernetes cluster
	if err := core.SyncGatewaysToK8s(ctx, appCtx); err != nil {
		return nil, err
	}

	res := toAppGatewayResponse(&gateway)
	return &res, nil
}

// DeleteAppGateway deletes a gateway
func DeleteAppGateway(ctx context.Context, id string) error {
	var gateway entities.AppGateway
	err := db.DB.First(&gateway, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("gateway not found")
		}
		return err
	}

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, gateway.AppID)
	if err != nil {
		return err
	}

	// Delete from Kubernetes first
	if err := core.DeleteGatewayFromK8s(ctx, appCtx, &gateway); err != nil {
		return err
	}
	// Delete from database
	if err := db.DB.Delete(&gateway).Error; err != nil {
		return err
	}

	return nil
}

// toAppGatewayResponse converts an AppGateway entity to a response model with snake_case JSON fields.
func toAppGatewayResponse(gw *entities.AppGateway) models.AppGatewayResponse {
	return models.AppGatewayResponse{
		ID:          gw.ID,
		AppID:       gw.AppID,
		Port:        gw.Port,
		Protocol:    gw.Protocol,
		Domain:      gw.Domain,
		Path:        gw.Path,
		GatewayPort: gw.GatewayPort,
		Exposed:     gw.Exposed,
		CertID:      gw.CertID,
		CreatedAt:   gw.CreatedAt,
		UpdatedAt:   gw.UpdatedAt,
	}
}

// GetGatewayWithApp loads a gateway along with its parent App, Env, and Cluster.
func GetGatewayWithApp(ctx context.Context, gatewayID string) (*entities.AppGateway, *models.AppContext, error) {
	var gateway entities.AppGateway
	err := db.DB.First(&gateway, "id = ?", gatewayID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("gateway not found")
		}
		return nil, nil, err
	}

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, gateway.AppID)
	if err != nil {
		return nil, nil, err
	}
	return &gateway, appCtx, nil
}
