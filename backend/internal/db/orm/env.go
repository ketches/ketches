package orm

import (
	"context"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

func GetEnvByID(ctx context.Context, envID string) (*entities.Env, app.Error) {
	env := &entities.Env{}
	if err := db.Instance().First(env, "id = ?", envID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Env not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return env, nil
}

func GetProjectIDByEnvID(ctx context.Context, envID string) (string, app.Error) {
	env := &entities.Env{}
	if err := db.Instance().Select("project_id").First(env, "id = ?", envID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "Env not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	return env.ProjectID, nil
}

func GetProjectRoleByEnvID(ctx context.Context, envID string) (string, app.Error) {
	userID := api.UserID(ctx)
	if userID == "" {
		return "", app.ErrNotAuthorized
	}

	entity := &models.ProjectMemberRole{}
	if err := db.Instance().Model(&entities.Env{}).Joins("JOIN project_members ON project_members.project_id = envs.project_id").Select("project_members.project_role").First(entity, "envs.id = ? AND project_members.user_id = ?", envID, userID).Error; err != nil {
		log.Printf("failed to get project role for env %s: %v", envID, err)
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "Env not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	if entity.ProjectRole == "" {
		return "", app.ErrPermissionDenied
	}

	return entity.ProjectRole, nil
}

func CountEnvApps(ctx context.Context, envID string) (int64, app.Error) {
	var count int64
	if err := db.Instance().Model(&entities.App{}).Where("env_id = ?", envID).Count(&count).Error; err != nil {
		return 0, app.ErrDatabaseOperationFailed
	}
	return count, nil
}
