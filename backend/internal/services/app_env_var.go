package services

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
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
		UUIDBase: entities.UUIDBase{ID: uuid.NewString()},
		AppID:    req.AppID,
		Key:      req.Key,
		Value:    req.Value,
	}
	if err := db.Instance().Create(entity).Error; err != nil {
		return nil, app.NewError(500, err.Error())
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
		return nil, app.NewError(404, "env var not found")
	}

	entity.Value = req.Value
	if err := db.Instance().Model(&entity).Update("value", req.Value).Error; err != nil {
		return nil, app.NewError(500, err.Error())
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
	if err := db.Instance().Where("id IN ?", req.EnvVarIDs).Delete(&entities.AppEnvVar{}).Error; err != nil {
		return app.NewError(500, err.Error())
	}
	return nil
}
