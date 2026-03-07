package services

import (
	"context"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

func ListDeletedApps(projectID string, userID string, page, pageSize int, search string) (int64, []models.RecycleBinAppRow, error) {
	var rows []models.RecycleBinAppRow
	var total int64
	query := db.DB.Table("apps").
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

	query := db.DB.Table("envs").
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

func BatchRestoreApps(appIDs []string) error {
	for _, appID := range appIDs {
		if err := RestoreApp(context.Background(), appID); err != nil {
			return err
		}
	}
	return nil
}

func BatchPermanentlyDeleteApps(appIDs []string) error {
	for _, appID := range appIDs {
		if err := PermanentlyDeleteApp(context.Background(), appID); err != nil {
			return err
		}
	}
	return nil
}

func BatchRestoreEnvs(envIDs []string) error {
	for _, envID := range envIDs {
		if err := RestoreEnv(envID); err != nil {
			return err
		}
	}
	return nil
}

func BatchPermanentlyDeleteEnvs(envIDs []string) error {
	for _, envID := range envIDs {
		if err := PermanentlyDeleteEnv(envID); err != nil {
			return err
		}
	}
	return nil
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

// BatchRestoreProjects restores multiple soft-deleted projects by ID.
func BatchRestoreProjects(ids []string) error {
	for _, id := range ids {
		if err := RestoreProject(id); err != nil {
			return err
		}
	}
	return nil
}

// BatchPermanentlyDeleteProjects permanently deletes multiple projects by ID.
func BatchPermanentlyDeleteProjects(ids []string) error {
	for _, id := range ids {
		if err := PermanentlyDeleteProject(id); err != nil {
			return err
		}
	}
	return nil
}
