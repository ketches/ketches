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
func ListAppGateways(appID string) ([]entities.AppGateway, error) {
	var gateways []entities.AppGateway
	err := db.DB.Where("app_id = ?", appID).Find(&gateways).Error
	return gateways, err
}

// CreateAppGateway creates a new gateway for an app
func CreateAppGateway(appID string, req *models.CreateGatewayRequest) (*entities.AppGateway, error) {
	entity := &entities.AppGateway{
		ID:          uuid.New(),
		AppID:       appID,
		Port:        req.Port,
		Protocol:    req.Protocol,
		Domain:      req.Domain,
		Path:        req.Path,
		GatewayPort: req.GatewayPort,
		Exposed:     req.Exposed,
		CertID:      req.CertID,
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

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.Preload("Env").Preload("Gateways").First(&app, "id = ?", appID).Error; err != nil {
		return nil, err
	}

	// Sync to Kubernetes cluster
	if err := core.SyncGatewayToK8s(context.Background(), &app, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// UpdateAppGateway updates an existing gateway
func UpdateAppGateway(id string, req *models.UpdateGatewayRequest) (*entities.AppGateway, error) {
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
	gateway.CertID = req.CertID

	// Save to database
	if err := db.DB.Save(&gateway).Error; err != nil {
		return nil, err
	}

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.Preload("Env").Preload("Gateways").First(&app, "id = ?", gateway.AppID).Error; err != nil {
		return nil, err
	}

	// Sync to Kubernetes cluster
	if err := core.SyncGatewayToK8s(context.Background(), &app, &gateway); err != nil {
		return nil, err
	}

	return &gateway, nil
}

// DeleteAppGateway deletes a gateway
func DeleteAppGateway(id string) error {
	var gateway entities.AppGateway
	err := db.DB.First(&gateway, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("gateway not found")
		}
		return err
	}

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.Preload("Env").Preload("Gateways").First(&app, "id = ?", gateway.AppID).Error; err != nil {
		return err
	}

	// Delete from Kubernetes first
	if err := core.DeleteGatewayFromK8s(context.Background(), &app, &gateway); err != nil {
		return err
	}

	// Delete from database
	if err := db.DB.Delete(&gateway).Error; err != nil {
		return err
	}

	return nil
}
