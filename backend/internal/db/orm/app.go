package orm

import (
	"context"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entity"
)

func AllAppEnvVars(appID string) ([]*entity.AppEnvVar, app.Error) {
	var result []*entity.AppEnvVar
	if err := db.Instance().Find(&result, "app_id = ?", appID).Error; err != nil {
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func GetAppByID(ctx context.Context, appID string) (*entity.App, app.Error) {
	result := &entity.App{}
	if err := db.Instance().First(result, "id = ?", appID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "App not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func UpdateAppDeployInfo(ctx context.Context, appID string, deployed bool, deployVersion string) app.Error {
	if err := db.Instance().Updates(&entity.App{
		UUIDBase: entity.UUIDBase{
			ID: appID,
		},
		Deployed:      deployed,
		DeployVersion: deployVersion,
		AuditBase: entity.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update app info on deploy: %v", err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}
