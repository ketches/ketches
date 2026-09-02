package services

import (
	"context"
	"errors"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"gorm.io/gorm"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// RecycleBinActor is the authenticated identity performing a recycle-bin action.
// System administrators bypass project membership checks; all other users must
// be an owner of every affected project.
type RecycleBinActor struct {
	UserID string
	Role   string
}

var (
	ErrRecycleBinAccessDenied     = errors.New("insufficient recycle bin permissions")
	ErrRecycleBinInvalidIDs       = errors.New("recycle bin resource IDs are required")
	ErrRecycleBinResourceNotFound = errors.New("recycle bin resource not found")
	ErrRecycleBinResourceActive   = errors.New("resource is not soft-deleted")
	ErrRecycleBinResourceDeleting = errors.New("resource is being permanently deleted")
	ErrRecycleBinParentDeleted    = errors.New("parent resource is soft-deleted")
	ErrRecycleBinActiveChildren   = errors.New("resource contains active children")
)

func normalizeRecycleBinIDs(ids []string) ([]string, error) {
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, ErrRecycleBinInvalidIDs
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil, ErrRecycleBinInvalidIDs
	}
	return unique, nil
}

func ensureRecycleBinOwner(tx *gorm.DB, projectID string, actor RecycleBinActor) error {
	if actor.Role == app.UserRoleAdmin {
		return nil
	}
	if strings.TrimSpace(actor.UserID) == "" {
		return ErrRecycleBinAccessDenied
	}

	var count int64
	if err := tx.Model(&entities.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND project_role = ?", projectID, actor.UserID, app.ProjectRoleOwner).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrRecycleBinAccessDenied
	}
	return nil
}

func recycleBinActorFromArgs(actors []RecycleBinActor) (RecycleBinActor, error) {
	if len(actors) != 1 {
		return RecycleBinActor{}, ErrRecycleBinAccessDenied
	}
	return actors[0], nil
}

func recycleBinActionError(resourceID string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return app.WrapErrorf(ErrRecycleBinResourceNotFound, "resource %s", resourceID)
	}
	return err
}

func ListDeletedApps(projectID string, userID string, page, pageSize int, search string) (int64, []models.RecycleBinAppRow, error) {
	var rows []models.RecycleBinAppRow
	var total int64
	query := db.DB.Unscoped().Table("apps").
		Select("apps.*, envs.name as env_name, projects.name as project_name, projects.slug as project_slug").
		Joins("JOIN envs ON apps.env_id = envs.id").
		Joins("JOIN projects ON envs.project_id = projects.id").
		Where("apps.deleted_at IS NOT NULL").
		Order("apps.deleted_at DESC")

	if projectID != "" {
		query = query.Where("envs.project_id = ?", projectID)
	} else if userID != "" {
		// Filter to projects where the user has non-viewer role (owner or developer)
		query = query.Joins("JOIN project_members ON project_members.project_id = envs.project_id").
			Where("project_members.user_id = ? AND project_members.project_role IN ('owner', 'developer')", userID)
	}

	if search != "" {
		query = query.Where("apps.name LIKE ? OR apps.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}

	return total, rows, nil
}

func ListDeletedEnvs(projectID string, userID string, page, pageSize int, search string) (int64, []models.RecycleBinEnvResponse, error) {
	var rows []models.RecycleBinEnvRow
	var total int64

	query := db.DB.Unscoped().Table("envs").
		Select("envs.*, projects.name as project_name, projects.slug as project_slug, clusters.name as cluster_name").
		Joins("JOIN projects ON envs.project_id = projects.id").
		Joins("JOIN clusters ON envs.cluster_id = clusters.id").
		Where("envs.deleted_at IS NOT NULL").
		Order("envs.deleted_at DESC")

	if projectID != "" {
		query = query.Where("envs.project_id = ?", projectID)
	} else if userID != "" {
		// Filter to projects where the user has non-viewer role (owner or developer)
		query = query.Joins("JOIN project_members ON project_members.project_id = envs.project_id").
			Where("project_members.user_id = ? AND project_members.project_role IN ('owner', 'developer')", userID)
	}

	if search != "" {
		query = query.Where("envs.name LIKE ? OR envs.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}

	// Convert RecycleBinEnvRow to RecycleBinEnvResponse
	result := []models.RecycleBinEnvResponse{}
	for _, row := range rows {
		result = append(result, models.RecycleBinEnvResponse{
			ID:               row.ID,
			Slug:             row.Slug,
			Name:             row.Name,
			Description:      row.Description,
			ProjectID:        row.ProjectID,
			ProjectName:      row.ProjectName,
			ProjectSlug:      row.ProjectSlug,
			ClusterID:        row.ClusterID,
			ClusterName:      row.ClusterName,
			ClusterNamespace: row.ClusterNamespace,
			DeletedAt:        row.DeletedAt.Time,
		})
	}

	return total, result, nil
}

func loadRecycleBinApp(tx *gorm.DB, appID string, actor RecycleBinActor) (*entities.App, error) {
	var application entities.App
	if err := tx.Unscoped().First(&application, "id = ?", appID).Error; err != nil {
		return nil, recycleBinActionError(appID, err)
	}
	if !application.DeletedAt.Valid {
		return nil, app.WrapErrorf(ErrRecycleBinResourceActive, "app %s", appID)
	}

	var env entities.Env
	if err := tx.Unscoped().Select("project_id").First(&env, "id = ?", application.EnvID).Error; err != nil {
		return nil, recycleBinActionError(appID, err)
	}
	if err := ensureRecycleBinOwner(tx, env.ProjectID, actor); err != nil {
		return nil, err
	}
	return &application, nil
}

func loadRecycleBinEnv(tx *gorm.DB, envID string, actor RecycleBinActor) (*entities.Env, error) {
	var env entities.Env
	if err := tx.Unscoped().First(&env, "id = ?", envID).Error; err != nil {
		return nil, recycleBinActionError(envID, err)
	}
	if !env.DeletedAt.Valid {
		return nil, app.WrapErrorf(ErrRecycleBinResourceActive, "environment %s", envID)
	}
	if err := ensureRecycleBinOwner(tx, env.ProjectID, actor); err != nil {
		return nil, err
	}
	return &env, nil
}

func BatchRestoreApps(appIDs []string, actors ...RecycleBinActor) error {
	actor, err := recycleBinActorFromArgs(actors)
	if err != nil {
		return err
	}
	ids, err := normalizeRecycleBinIDs(appIDs)
	if err != nil {
		return err
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, appID := range ids {
			if _, err := loadRecycleBinApp(tx, appID, actor); err != nil {
				return err
			}
		}
		for _, appID := range ids {
			if err := restoreAppTx(tx, appID); err != nil {
				return err
			}
		}
		return nil
	})
}

func BatchPermanentlyDeleteApps(appIDs []string, actors ...RecycleBinActor) error {
	actor, err := recycleBinActorFromArgs(actors)
	if err != nil {
		return err
	}
	ids, err := normalizeRecycleBinIDs(appIDs)
	if err != nil {
		return err
	}

	cleanupContexts := make([]*models.AppContext, 0, len(ids))
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		targets := make([]recycleBinDeletionTarget, 0, len(ids))
		for _, appID := range ids {
			if _, err := loadRecycleBinApp(tx, appID, actor); err != nil {
				return err
			}
			cleanupContext, err := loadDeletedAppCleanupContext(tx, appID)
			if err != nil {
				return err
			}
			cleanupContexts = append(cleanupContexts, cleanupContext)
			targets = append(targets, newRecycleBinDeletionTarget(
				recycleBinResourceApp, appID, &entities.App{}, "app",
			))
		}
		return claimRecycleBinDeletionTargets(tx, targets...)
	}); err != nil {
		return err
	}
	for _, cleanupContext := range cleanupContexts {
		if err := deleteAppK8sResources(context.Background(), cleanupContext, false); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, appID := range ids {
			if _, err := loadRecycleBinApp(tx, appID, actor); err != nil {
				return err
			}
		}
		for _, appID := range ids {
			if err := permanentlyDeleteAppTx(context.Background(), tx, appID); err != nil {
				return err
			}
		}
		return nil
	})
}

func BatchRestoreEnvs(envIDs []string, actors ...RecycleBinActor) error {
	actor, err := recycleBinActorFromArgs(actors)
	if err != nil {
		return err
	}
	ids, err := normalizeRecycleBinIDs(envIDs)
	if err != nil {
		return err
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, envID := range ids {
			if _, err := loadRecycleBinEnv(tx, envID, actor); err != nil {
				return err
			}
		}
		for _, envID := range ids {
			if err := restoreEnvTx(tx, envID); err != nil {
				return err
			}
		}
		return nil
	})
}

func BatchPermanentlyDeleteEnvs(envIDs []string, actors ...RecycleBinActor) error {
	actor, err := recycleBinActorFromArgs(actors)
	if err != nil {
		return err
	}
	ids, err := normalizeRecycleBinIDs(envIDs)
	if err != nil {
		return err
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		targets := make([]recycleBinDeletionTarget, 0, len(ids))
		for _, envID := range ids {
			if _, err := loadRecycleBinEnv(tx, envID, actor); err != nil {
				return err
			}
			targets = append(targets, newRecycleBinDeletionTarget(
				recycleBinResourceEnvironment, envID, &entities.Env{}, "environment",
			))
		}
		if err := claimRecycleBinDeletionTargets(tx, targets...); err != nil {
			return err
		}
		for _, envID := range ids {
			if err := validateEnvPermanentDeletionTx(tx, envID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	for _, envID := range ids {
		if err := cleanupDeletedEnvNamespace(context.Background(), envID); err != nil {
			return err
		}
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, envID := range ids {
			if _, err := loadRecycleBinEnv(tx, envID, actor); err != nil {
				return err
			}
		}
		for _, envID := range ids {
			if err := permanentlyDeleteEnvTx(context.Background(), tx, envID); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListDeletedProjects returns soft-deleted projects, paginated, with optional search.
// If userID is non-empty, only projects where the user has a non-viewer role (owner/developer) are returned.
func ListDeletedProjects(userID string, page, pageSize int, search string) (int64, []models.RecycleBinProjectResponse, error) {
	var projects []entities.Project
	var total int64
	query := db.DB.Unscoped().Model(&entities.Project{}).Where("projects.deleted_at IS NOT NULL").Order("deleted_at DESC")

	if userID != "" {
		// Filter to projects where the user has non-viewer role (owner or developer)
		query = query.Joins("JOIN project_members ON project_members.project_id = projects.id").
			Where("project_members.user_id = ? AND project_members.project_role IN ('owner', 'developer')", userID)
	}

	if search != "" {
		query = query.Where("projects.name LIKE ? OR projects.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&projects).Error; err != nil {
		return 0, nil, err
	}

	result := []models.RecycleBinProjectResponse{}
	for _, p := range projects {
		result = append(result, models.RecycleBinProjectResponse{
			ID:          p.ID,
			Slug:        p.Slug,
			Name:        p.Name,
			Description: p.Description,
			DeletedAt:   p.DeletedAt.Time,
		})
	}
	return total, result, nil
}

func loadRecycleBinProject(tx *gorm.DB, projectID string, actor RecycleBinActor) (*entities.Project, error) {
	var project entities.Project
	if err := tx.Unscoped().First(&project, "id = ?", projectID).Error; err != nil {
		return nil, recycleBinActionError(projectID, err)
	}
	if !project.DeletedAt.Valid {
		return nil, app.WrapErrorf(ErrRecycleBinResourceActive, "project %s", projectID)
	}
	if err := ensureRecycleBinOwner(tx, project.ID, actor); err != nil {
		return nil, err
	}
	return &project, nil
}

// BatchRestoreProjects restores multiple soft-deleted projects by ID.
func BatchRestoreProjects(projectIDs []string, actors ...RecycleBinActor) error {
	actor, err := recycleBinActorFromArgs(actors)
	if err != nil {
		return err
	}
	ids, err := normalizeRecycleBinIDs(projectIDs)
	if err != nil {
		return err
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, projectID := range ids {
			if _, err := loadRecycleBinProject(tx, projectID, actor); err != nil {
				return err
			}
		}
		for _, projectID := range ids {
			if err := restoreProjectTx(tx, projectID); err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchPermanentlyDeleteProjects permanently deletes multiple projects by ID.
func BatchPermanentlyDeleteProjects(projectIDs []string, actors ...RecycleBinActor) error {
	actor, err := recycleBinActorFromArgs(actors)
	if err != nil {
		return err
	}
	ids, err := normalizeRecycleBinIDs(projectIDs)
	if err != nil {
		return err
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		targets := make([]recycleBinDeletionTarget, 0, len(ids))
		for _, projectID := range ids {
			if _, err := loadRecycleBinProject(tx, projectID, actor); err != nil {
				return err
			}
			targets = append(targets, newRecycleBinDeletionTarget(
				recycleBinResourceProject, projectID, &entities.Project{}, "project",
			))
		}
		if err := claimRecycleBinDeletionTargets(tx, targets...); err != nil {
			return err
		}
		for _, projectID := range ids {
			if err := validateProjectPermanentDeletionTx(tx, projectID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	for _, projectID := range ids {
		if err := cleanupProjectNamespaces(context.Background(), projectID); err != nil {
			return err
		}
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, projectID := range ids {
			if _, err := loadRecycleBinProject(tx, projectID, actor); err != nil {
				return err
			}
		}
		for _, projectID := range ids {
			if err := permanentlyDeleteProjectTx(tx, projectID); err != nil {
				return err
			}
		}
		return nil
	})
}

func ListDeletedUsers(page, pageSize int, search string) (int64, []models.RecycleBinUserResponse, error) {
	var users []entities.User
	var total int64

	query := db.DB.Unscoped().Model(&entities.User{}).Where("users.deleted_at IS NOT NULL").Order("deleted_at DESC")

	if search != "" {
		query = query.Where("users.username LIKE ? OR users.fullname LIKE ? OR users.email LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return 0, nil, err
	}

	result := make([]models.RecycleBinUserResponse, 0, len(users))
	for _, u := range users {
		result = append(result, models.RecycleBinUserResponse{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			Fullname:  u.Fullname,
			Role:      u.Role,
			DeletedAt: u.DeletedAt.Time,
		})
	}

	return total, result, nil
}

func BatchRestoreUsers(userIDs []string) error {
	ids, err := normalizeRecycleBinIDs(userIDs)
	if err != nil {
		return err
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if err := restoreUserTx(tx, id); err != nil {
				if errors.Is(err, ErrRecycleBinResourceActive) {
					return app.WrapError("cannot restore active user", err)
				}
				return err
			}
		}
		return nil
	})
}

func BatchPermanentlyDeleteUsers(ids []string) error {
	for _, id := range ids {
		if err := PermanentlyDeleteUser(id); err != nil {
			return err
		}
	}
	return nil
}
