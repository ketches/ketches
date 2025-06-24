package orm

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entity"
)

func GetProjectSlugByID(ctx context.Context, projectID string) (string, app.Error) {
	project := &entity.Project{}
	if err := db.Instance().Select("slug").First(project, "id = ?", projectID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "Project not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	return project.Slug, nil
}
