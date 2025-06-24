package orm

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entity"
)

func GetEnvByID(ctx context.Context, envID string) (*entity.Env, app.Error) {
	env := &entity.Env{}
	if err := db.Instance().First(env, "id = ?", envID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Env not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return env, nil
}

func GetProjectIDByEnvID(ctx context.Context, envID string) (string, app.Error) {
	env := &entity.Env{}
	if err := db.Instance().Select("project_id").First(env, "id = ?", envID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "Env not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	return env.ProjectID, nil
}

func CountEnvApps(ctx context.Context, envID string) (int64, app.Error) {
	var count int64
	if err := db.Instance().Model(&entity.App{}).Where("env_id = ?", envID).Count(&count).Error; err != nil {
		return 0, app.ErrDatabaseOperationFailed
	}
	return count, nil
}
