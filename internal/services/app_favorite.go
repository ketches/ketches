package services

import (
	"context"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// ListFavoriteApps returns full app details for all apps favorited by a user within an environment.
func ListFavoriteApps(c context.Context, userID, envID string) ([]models.AppResponse, error) {
	var apps []entities.App
	if err := db.DB.Model(&entities.App{}).Preload("Env.Cluster").
		Joins("JOIN app_favorites ON app_favorites.app_id = apps.id").
		Where("app_favorites.user_id = ? AND app_favorites.env_id = ?", userID, envID).
		Order("app_favorites.created_at DESC").
		Find(&apps).Error; err != nil {
		return nil, err
	}
	result := make([]models.AppResponse, 0, len(apps))
	for i := range apps {
		result = append(result, ToAppResponse(c, &apps[i]))
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
