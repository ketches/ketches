package services

import (
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
)

// ListFavoriteApps returns all favorite records for a user.
func ListFavoriteApps(userID string) ([]entities.AppFavorite, error) {
	var favorites []entities.AppFavorite
	if err := db.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&favorites).Error; err != nil {
		return nil, err
	}
	return favorites, nil
}

// IsFavoriteApp returns true if the user has favorited the given app.
func IsFavoriteApp(userID, appID string) bool {
	var count int64
	db.DB.Model(&entities.AppFavorite{}).Where("user_id = ? AND app_id = ?", userID, appID).Count(&count)
	return count > 0
}

// AddFavoriteApp adds an app to the user's favorites (idempotent).
func AddFavoriteApp(userID, appID string) (*entities.AppFavorite, error) {
	if IsFavoriteApp(userID, appID) {
		var existing entities.AppFavorite
		db.DB.Where("user_id = ? AND app_id = ?", userID, appID).First(&existing)
		return &existing, nil
	}
	fav := &entities.AppFavorite{
		Base:   entities.Base{ID: uuid.New()},
		UserID: userID,
		AppID:  appID,
	}
	if err := db.DB.Create(fav).Error; err != nil {
		return nil, err
	}
	return fav, nil
}

// RemoveFavoriteApp removes an app from the user's favorites.
func RemoveFavoriteApp(userID, appID string) error {
	return db.DB.Where("user_id = ? AND app_id = ?", userID, appID).Delete(&entities.AppFavorite{}).Error
}
