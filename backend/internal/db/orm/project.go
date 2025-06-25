package orm

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

func GetProjectSlugByID(ctx context.Context, projectID string) (string, app.Error) {
	project := &entities.Project{}
	if err := db.Instance().Select("slug").First(project, "id = ?", projectID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "Project not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	return project.Slug, nil
}

func GetProjectRoleByProjectID(ctx context.Context, projectID string) (string, app.Error) {
	userID := api.UserID(ctx)
	if userID == "" {
		return "", app.ErrNotAuthorized
	}

	entity := &models.ProjectMemberRole{}
	if err := db.Instance().Model(&entities.Project{}).Joins("JOIN project_members ON project_members.project_id = projects.id").Select("project_members.project_role").First(entity, "projects.id = ? AND project_members.user_id = ?", projectID, userID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "Project not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	if entity.ProjectRole == "" {
		return "", app.ErrPermissionDenied
	}

	return entity.ProjectRole, nil
}
