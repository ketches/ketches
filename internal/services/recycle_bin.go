package services

import (
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

func ListDeletedApps(projectID string, page, pageSize int, search string) (int64, []models.RecycleBinAppResponse, error) {
	var apps []entities.App
	var total int64
	query := db.DB.Unscoped().Model(&entities.App{}).Where("apps.deleted_at IS NOT NULL").Order("deleted_at DESC")

	if projectID != "" {
		query = query.Joins("JOIN envs ON apps.env_id = envs.id").
			Where("envs.project_id = ?", projectID)
	}

	if search != "" {
		query = query.Where("apps.name LIKE ? OR apps.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := query.Preload("Env.Project").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&apps).Error; err != nil {
		return 0, nil, err
	}

	var result []models.RecycleBinAppResponse
	for _, app := range apps {
		result = append(result, models.RecycleBinAppResponse{
			ID:             app.ID,
			Slug:           app.Slug,
			Name:           app.Name,
			Description:    app.Description,
			EnvID:          app.EnvID,
			EnvName:        app.Env.Name,
			ProjectID:      app.Env.ProjectID,
			ProjectName:    app.Env.Project.Name,
			AppType:        app.AppType,
			ContainerImage: app.ContainerImage,
			DeletedAt:      app.DeletedAt.Time,
		})
	}

	return total, result, nil
}

func ListDeletedEnvs(projectID string, page, pageSize int, search string) (int64, []models.RecycleBinEnvResponse, error) {
	var envs []entities.Env
	var total int64
	query := db.DB.Unscoped().Model(&entities.Env{}).Where("envs.deleted_at IS NOT NULL").Order("deleted_at DESC")

	if projectID != "" {
		query = query.Where("envs.project_id = ?", projectID)
	}

	if search != "" {
		query = query.Where("envs.name LIKE ? OR envs.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := query.Preload("Project").Preload("Cluster").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&envs).Error; err != nil {
		return 0, nil, err
	}

	var result []models.RecycleBinEnvResponse
	for _, env := range envs {
		clusterName := ""
		if env.Cluster.ID != "" {
			clusterName = env.Cluster.Name
		}
		result = append(result, models.RecycleBinEnvResponse{
			ID:               env.ID,
			Slug:             env.Slug,
			Name:             env.Name,
			Description:      env.Description,
			ProjectID:        env.ProjectID,
			ProjectName:      env.Project.Name,
			ClusterID:        env.ClusterID,
			ClusterName:      clusterName,
			ClusterNamespace: env.ClusterNamespace,
			DeletedAt:        env.DeletedAt.Time,
		})
	}

	return total, result, nil
}

func BatchRestoreApps(appIDs []string) error {
	for _, appID := range appIDs {
		if err := RestoreApp(appID); err != nil {
			return err
		}
	}
	return nil
}

func BatchPermanentlyDeleteApps(appIDs []string) error {
	for _, appID := range appIDs {
		if err := PermanentlyDeleteApp(appID); err != nil {
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
