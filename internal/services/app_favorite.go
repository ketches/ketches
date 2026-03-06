package services

import (
	"context"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// ListFavoriteApps returns a list of apps favorited by the user in the given environment.
// Uses explicit JOINs on envs and clusters to avoid N+1 queries.
func ListFavoriteApps(c context.Context, userID, envID string) ([]models.AppResponse, error) {
	var rows []models.AppListRow
	if err := db.DB.Table("apps").
		Select(`apps.id, apps.slug, apps.name, apps.description, apps.env_id,
			apps.app_type, apps.code_repository_id, apps.container_image,
			apps.container_command, apps.registry_username, apps.registry_password,
			apps.replicas, apps.request_cpu, apps.request_memory,
			apps.limit_cpu, apps.limit_memory, apps.deploy_status, apps.created_at,
			envs.name AS env_name, envs.slug AS env_slug,
			envs.cluster_id, envs.cluster_namespace, envs.is_build_env,
			clusters.name AS cluster_name`).
		Joins("JOIN app_favorites ON app_favorites.app_id = apps.id").
		Joins("JOIN envs ON envs.id = apps.env_id").
		Joins("JOIN clusters ON clusters.id = envs.cluster_id").
		Where("app_favorites.user_id = ? AND app_favorites.env_id = ?", userID, envID).
		Where("apps.deleted_at IS NULL").
		Order("app_favorites.created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]models.AppResponse, 0, len(rows))
	for i := range rows {
		result = append(result, ToAppListResponse(c, &rows[i]))
	}
	return result, nil
}

// IsFavoriteApp returns true if the user has favorited the given app in the given env.
func IsFavoriteApp(userID, appID, envID string) bool {
	var count int64
	db.DB.Model(&entities.AppFavorite{}).Where("user_id = ? AND app_id = ? AND env_id = ?", userID, appID, envID).Count(&count)
	return count > 0
}

// AddFavoriteApp adds an app to the user's favorites for an environment (idempotent).
func AddFavoriteApp(userID, appID, envID string) (*entities.AppFavorite, error) {
	if IsFavoriteApp(userID, appID, envID) {
		var existing entities.AppFavorite
		db.DB.Where("user_id = ? AND app_id = ? AND env_id = ?", userID, appID, envID).First(&existing)
		return &existing, nil
	}
	fav := &entities.AppFavorite{
		ID:     uuid.New(),
		UserID: userID,
		EnvID:  envID,
		AppID:  appID,
	}
	if err := db.DB.Create(fav).Error; err != nil {
		return nil, err
	}
	return fav, nil
}

// RemoveFavoriteApp removes an app from the user's favorites for an environment.
func RemoveFavoriteApp(userID, appID, envID string) error {
	return db.DB.Where("user_id = ? AND app_id = ? AND env_id = ?", userID, appID, envID).Delete(&entities.AppFavorite{}).Error
}
