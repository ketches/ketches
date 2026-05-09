package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

var ErrInvalidGatewayCertificate = errors.New("invalid gateway certificate")

type gatewayCertificateError struct {
	message string
}

func (e gatewayCertificateError) Error() string {
	return ErrInvalidGatewayCertificate.Error() + ": " + e.message
}

func (e gatewayCertificateError) Unwrap() error {
	return ErrInvalidGatewayCertificate
}

func newGatewayCertificateError(message string) error {
	return gatewayCertificateError{message: message}
}

// ListAppGateways returns all gateways for a given app
func ListAppGateways(appID string) ([]models.AppGatewayResponse, error) {
	var gateways []entities.AppGateway
	err := db.DB.Where("app_id = ?", appID).Find(&gateways).Error
	if err != nil {
		return nil, err
	}

	var gatewayHost, appSlug, namespace string
	if len(gateways) > 0 {
		if appCtx, err := GetAppContext(context.Background(), appID); err == nil {
			gatewayHost = appCtx.EnvContext.Cluster.GatewayHost
			appSlug = appCtx.App.Slug
			namespace = appCtx.EnvContext.Env.ClusterNamespace
		}
	}

	result := make([]models.AppGatewayResponse, 0, len(gateways))
	for _, gw := range gateways {
		resp := toAppGatewayResponse(&gw)
		resp.GatewayHost = gatewayHost
		if appSlug != "" {
			resp.InternalAddress = fmt.Sprintf("%s.%s:%d", appSlug, namespace, gw.Port)
		}
		result = append(result, resp)
	}
	return result, nil
}

// CreateAppGateway creates a new gateway for an app
func CreateAppGateway(ctx context.Context, appID string, req *models.CreateGatewayRequest) (*models.AppGatewayResponse, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}
	if err := validateGatewayCertificateReference(appCtx, req.Protocol, req.Exposed, req.CertID); err != nil {
		return nil, err
	}

	entity := &entities.AppGateway{
		ID:          uuid.New(),
		AppID:       appID,
		Port:        req.Port,
		Protocol:    req.Protocol,
		Domain:      req.Domain,
		Path:        req.Path,
		GatewayPort: req.GatewayPort,
		ServiceType: req.ServiceType,
		Exposed:     req.Exposed,
	}
	if req.ServiceType == "NodePort" && req.NodePort != 0 {
		entity.NodePort = req.NodePort
	}
	if req.CertID != "" && req.Exposed && req.Protocol == "https" {
		certID := req.CertID
		entity.CertID = &certID
	}

	var existing entities.AppGateway
	err = db.DB.Where("app_id = ? AND port = ? AND protocol = ?", appID, req.Port, req.Protocol).First(&existing).Error
	if err == nil {
		return nil, errors.New("gateway with this port and protocol already exists for this app")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create in database.
	if err := db.DB.Create(entity).Error; err != nil {
		return nil, err
	}

	// Fetch full app context from DB.
	appCtx, err = GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	// Sync to Kubernetes cluster.
	if err := core.SyncGatewaysToK8s(ctx, appCtx); err != nil {
		return nil, err
	}

	// Read back actual NodePorts assigned by K8s and persist.
	if nodePorts, err := core.ReadNodePortsFromK8s(ctx, appCtx); err == nil {
		if np, ok := nodePorts[entity.Port]; ok && entity.NodePort == 0 {
			entity.NodePort = np
			db.DB.Model(entity).Update("node_port", np)
		}
	}

	res := toAppGatewayResponse(entity)
	res.GatewayHost = appCtx.EnvContext.Cluster.GatewayHost
	res.InternalAddress = fmt.Sprintf("%s.%s:%d", appCtx.App.Slug, appCtx.EnvContext.Env.ClusterNamespace, entity.Port)
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

	appCtx, err := GetAppContext(ctx, gateway.AppID)
	if err != nil {
		return nil, err
	}
	if err := validateGatewayCertificateReference(appCtx, req.Protocol, req.Exposed, req.CertID); err != nil {
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

	// Update fields.
	gateway.Port = req.Port
	gateway.Protocol = req.Protocol
	gateway.Domain = req.Domain
	gateway.Path = req.Path
	gateway.GatewayPort = req.GatewayPort
	gateway.ServiceType = req.ServiceType
	gateway.Exposed = req.Exposed
	if req.ServiceType == "NodePort" && req.NodePort != 0 {
		gateway.NodePort = req.NodePort
	} else if req.ServiceType != "NodePort" {
		gateway.NodePort = 0
	}
	if req.CertID != "" && req.Exposed && req.Protocol == "https" {
		certID := req.CertID
		gateway.CertID = &certID
	} else {
		gateway.CertID = nil
	}

	// Save to database.
	if err := db.DB.Save(&gateway).Error; err != nil {
		return nil, err
	}

	// Fetch full app context from DB.
	appCtx, err = GetAppContext(ctx, gateway.AppID)
	if err != nil {
		return nil, err
	}

	// Sync to Kubernetes cluster.
	if err := core.SyncGatewaysToK8s(ctx, appCtx); err != nil {
		return nil, err
	}

	// Read back actual NodePorts assigned by K8s and persist.
	if nodePorts, err := core.ReadNodePortsFromK8s(ctx, appCtx); err == nil {
		if np, ok := nodePorts[gateway.Port]; ok && gateway.NodePort == 0 {
			gateway.NodePort = np
			db.DB.Model(&gateway).Update("node_port", np)
		}
	}

	res := toAppGatewayResponse(&gateway)
	res.GatewayHost = appCtx.EnvContext.Cluster.GatewayHost
	res.InternalAddress = fmt.Sprintf("%s.%s:%d", appCtx.App.Slug, appCtx.EnvContext.Env.ClusterNamespace, gateway.Port)
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

func validateGatewayCertificateReference(appCtx *models.AppContext, protocol string, exposed bool, certID string) error {
	if appCtx == nil || !exposed || !strings.EqualFold(protocol, "https") {
		return nil
	}

	trimmedCertID := strings.TrimSpace(certID)
	if trimmedCertID == "" {
		return newGatewayCertificateError("certificate is required for HTTPS public access")
	}

	certificate, err := GetCertificate(trimmedCertID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newGatewayCertificateError("selected certificate was not found")
		}
		return err
	}
	if certificate.ClusterID != appCtx.EnvContext.Env.ClusterID {
		return newGatewayCertificateError("selected certificate does not belong to the current cluster")
	}
	if certificate.Scope == "env" {
		if certificate.EnvID == nil || strings.TrimSpace(*certificate.EnvID) != appCtx.EnvContext.Env.ID {
			return newGatewayCertificateError("selected certificate is not available in this environment")
		}
		return nil
	}
	if certificate.Scope != "cluster" {
		return newGatewayCertificateError("selected certificate has an unsupported scope")
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
		ServiceType: gw.ServiceType,
		NodePort:    gw.NodePort,
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
