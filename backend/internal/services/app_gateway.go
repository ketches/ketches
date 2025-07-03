package services

import (
	"context"
	"log"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/models"
)

type AppGatewayService interface {
	ListAppGateways(ctx context.Context, req *models.ListAppGatewaysRequest) ([]*models.AppGatewayModel, app.Error)
	CreateAppGateway(ctx context.Context, req *models.CreateAppGatewayRequest) (*models.AppGatewayModel, app.Error)
	UpdateAppGateway(ctx context.Context, req *models.UpdateAppGatewayRequest) (*models.AppGatewayModel, app.Error)
	DeleteAppGateways(ctx context.Context, req *models.DeleteAppGatewaysRequest) app.Error
}

type appGatewayService struct {
	Service
}

var appGatewayServiceInstance = &appGatewayService{
	Service: LoadService(),
}

func NewAppGatewayService() AppGatewayService {
	return appGatewayServiceInstance
}

func (s *appGatewayService) ListAppGateways(ctx context.Context, req *models.ListAppGatewaysRequest) ([]*models.AppGatewayModel, app.Error) {
	gateways := make([]*entities.AppGateway, 0)
	if err := db.Instance().Where("app_id = ?", req.AppID).Find(&gateways).Error; err != nil {
		log.Printf("failed to list app gateways: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	result := make([]*models.AppGatewayModel, 0, len(gateways))
	for _, gateway := range gateways {
		result = append(result, &models.AppGatewayModel{
			GatewayID:   gateway.ID,
			Port:        gateway.Port,
			Protocol:    gateway.Protocol,
			Domain:      gateway.Domain,
			Path:        gateway.Path,
			CertID:      gateway.CertID,
			GatewayPort: gateway.GatewayPort,
			Exposed:     gateway.Exposed,
			AppID:       gateway.AppID,
		})
	}

	return result, nil
}

func (s *appGatewayService) CreateAppGateway(ctx context.Context, req *models.CreateAppGatewayRequest) (*models.AppGatewayModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	gateway := &entities.AppGateway{
		AppID:       req.AppID,
		Port:        req.Port,
		Protocol:    req.Protocol,
		Domain:      req.Domain,
		Path:        req.Path,
		CertID:      req.CertID,
		GatewayPort: req.GatewayPort,
		Exposed:     req.Exposed,
		EnvID:       appEntity.EnvID,
		ProjectID:   appEntity.ProjectID,
		AuditBase: entities.AuditBase{
			CreatedBy: api.UserID(ctx),
			UpdatedBy: api.UserID(ctx),
		},
	}

	if err := db.Instance().Create(gateway).Error; err != nil {
		log.Printf("failed to create app gateway: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.AppGatewayModel{
		GatewayID:   gateway.ID,
		Port:        gateway.Port,
		Protocol:    gateway.Protocol,
		Domain:      gateway.Domain,
		Path:        gateway.Path,
		CertID:      gateway.CertID,
		GatewayPort: gateway.GatewayPort,
		Exposed:     gateway.Exposed,
		AppID:       gateway.AppID,
	}, nil
}

func (s *appGatewayService) UpdateAppGateway(ctx context.Context, req *models.UpdateAppGatewayRequest) (*models.AppGatewayModel, app.Error) {
	gateway, err := orm.GetAppGatewayByIDs(ctx, req.GatewayID)
	if err != nil {
		return nil, err
	}

	gateway.Port = req.Port
	gateway.Protocol = req.Protocol
	gateway.Domain = req.Domain
	gateway.Path = req.Path
	gateway.CertID = req.CertID
	gateway.GatewayPort = req.GatewayPort
	gateway.Exposed = req.Exposed
	gateway.UpdatedBy = api.UserID(ctx)

	if err := db.Instance().Save(gateway).Error; err != nil {
		log.Printf("failed to update app gateway: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.AppGatewayModel{
		GatewayID:   gateway.ID,
		Port:        gateway.Port,
		Protocol:    gateway.Protocol,
		Domain:      gateway.Domain,
		Path:        gateway.Path,
		CertID:      gateway.CertID,
		GatewayPort: gateway.GatewayPort,
		Exposed:     gateway.Exposed,
		AppID:       gateway.AppID,
	}, nil
}

func (s *appGatewayService) DeleteAppGateways(ctx context.Context, req *models.DeleteAppGatewaysRequest) app.Error {
	if len(req.GatewayIDs) == 0 {
		return nil
	}

	if err := db.Instance().Delete(&entities.AppGateway{}, req.GatewayIDs).Error; err != nil {
		log.Printf("failed to delete app gateways: %v", err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}
