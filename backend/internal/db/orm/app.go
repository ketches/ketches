package orm

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/spf13/cast"
)

func GetAppByID(ctx context.Context, appID string) (*entities.App, app.Error) {
	result := &entities.App{}
	if err := db.Instance().First(result, "id = ?", appID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "App not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func GetProjectIDByAppID(ctx context.Context, appID string) (string, app.Error) {
	entity := &entities.App{}
	if err := db.Instance().Select("project_id").First(entity, "id = ?", appID).Error; err != nil {
		log.Printf("failed to get project ID for app %s: %v", appID, err)
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "App not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	return entity.ProjectID, nil
}

func GetProjectRoleByAppID(ctx context.Context, appID string) (string, app.Error) {
	userID := api.UserID(ctx)
	if userID == "" {
		return "", app.ErrNotAuthorized
	}

	entity := &models.ProjectMemberRole{}
	if err := db.Instance().Model(&entities.App{}).Joins("JOIN project_members ON project_members.project_id = apps.project_id").Select("project_members.project_role").First(entity, "apps.id = ? AND project_members.user_id = ?", appID, userID).Error; err != nil {
		log.Printf("failed to get project role for app %s: %v", appID, err)
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "App not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	if entity.ProjectRole == "" {
		return "", app.ErrPermissionDenied
	}

	return entity.ProjectRole, nil
}

func UpdateAppEdition(ctx context.Context, appID string) app.Error {
	if err := db.Instance().Updates(&entities.App{
		UUIDBase: entities.UUIDBase{
			ID: appID,
		},
		Edition: cast.ToString(time.Now().UnixMilli()),
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update app edition: %v", err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func AllAppEnvVars(appID string) ([]*entities.AppEnvVar, app.Error) {
	var result []*entities.AppEnvVar
	if err := db.Instance().Find(&result, "app_id = ?", appID).Error; err != nil {
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func AllAppVolumes(appID string) ([]*entities.AppVolume, app.Error) {
	var result []*entities.AppVolume
	if err := db.Instance().Find(&result, "app_id = ?", appID).Error; err != nil {
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func AllAppGateways(appID string) ([]*entities.AppGateway, app.Error) {
	var result []*entities.AppGateway
	if err := db.Instance().Find(&result, "app_id = ?", appID).Error; err != nil {
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}
