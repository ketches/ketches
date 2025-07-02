package services

import (
	"context"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

type AppEnvVarService interface {
	ListAppEnvVars(ctx context.Context, req *models.ListAppEnvVarsRequest) ([]*models.AppEnvVarModel, app.Error)
	CreateAppEnvVar(ctx context.Context, req *models.CreateAppEnvVarRequest) (*models.AppEnvVarModel, app.Error)
	UpdateAppEnvVar(ctx context.Context, req *models.UpdateAppEnvVarRequest) (*models.AppEnvVarModel, app.Error)
	DeleteAppEnvVars(ctx context.Context, req *models.DeleteAppEnvVarsRequest) app.Error
}

type appEnvVarService struct {
	Service
}

var appEnvVarServiceInstance = &appEnvVarService{
	Service: LoadService(),
}

func NewAppEnvVarService() AppEnvVarService {
	return appEnvVarServiceInstance
}

// ListAppEnvVars returns all env vars for an app
func (s *appEnvVarService) ListAppEnvVars(ctx context.Context, req *models.ListAppEnvVarsRequest) ([]*models.AppEnvVarModel, app.Error) {
	var result []*models.AppEnvVarModel
	var total int64
	if err := db.Instance().Model(&entities.AppEnvVar{}).
		Where("app_id = ?", req.AppID).
		Count(&total).
		Find(&result).Error; err != nil {
		log.Printf("failed to list env vars for app %s: %v", req.AppID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

// CreateAppEnvVar creates a new env var for an app
func (s *appEnvVarService) CreateAppEnvVar(ctx context.Context, req *models.CreateAppEnvVarRequest) (*models.AppEnvVarModel, app.Error) {
	entity := &entities.AppEnvVar{
		UUIDBase: entities.UUIDBase{ID: uuid.New()},
		AppID:    req.AppID,
		Key:      req.Key,
		Value:    req.Value,
		AuditBase: entities.AuditBase{
			CreatedBy: api.UserID(ctx),
			UpdatedBy: api.UserID(ctx),
		},
	}
	if err := db.Instance().Create(entity).Error; err != nil {
		log.Printf("failed to create app env var for app %s: %v", req.AppID, err)
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusBadRequest, "env var key already exists")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after creating env var for app %s: %v", req.AppID, err)
	}

	return &models.AppEnvVarModel{
		EnvVarID: entity.ID,
		Key:      entity.Key,
		Value:    entity.Value,
		AppID:    entity.AppID,
	}, nil
}

func (s *appEnvVarService) UpdateAppEnvVar(ctx context.Context, req *models.UpdateAppEnvVarRequest) (*models.AppEnvVarModel, app.Error) {
	var entity entities.AppEnvVar
	if err := db.Instance().First(&entity, "id = ?", req.EnvVarID).Error; err != nil {
		log.Printf("failed to find app env var %s: %v", req.EnvVarID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "env var not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	entity.Value = req.Value
	if err := db.Instance().Model(&entity).Select("Value", "UpdatedBy").Updates(&entities.AppEnvVar{
		Value: req.Value,
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update app env var %s: %v", req.EnvVarID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after updating env var for app %s: %v", req.AppID, err)
	}

	return &models.AppEnvVarModel{
		EnvVarID: entity.ID,
		Key:      entity.Key,
		Value:    entity.Value,
		AppID:    entity.AppID,
	}, nil
}

func (s *appEnvVarService) DeleteAppEnvVars(ctx context.Context, req *models.DeleteAppEnvVarsRequest) app.Error {
	if len(req.EnvVarIDs) == 0 {
		return nil
	}

	if err := db.Instance().Delete(&entities.AppEnvVar{}, req.EnvVarIDs).Error; err != nil {
		log.Printf("failed to delete app env vars: %v", err)
		return app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after deleting env var for app %s: %v", req.AppID, err)
	}

	return nil
}
