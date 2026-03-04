package services

import (
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
)

// ListFavoriteApps returns all favorite records for a user within an environment.
func ListFavoriteApps(userID, envID string) ([]entities.AppFavorite, error) {
	var favorites []entities.AppFavorite
	if err := db.DB.Where("user_id = ? AND env_id = ?", userID, envID).Order("created_at DESC").Find(&favorites).Error; err != nil {
		return nil, err
	}
	return favorites, nil
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
