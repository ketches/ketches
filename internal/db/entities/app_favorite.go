package entities

import "time"

// AppFavorite records a user's favorite app within an environment.
type AppFavorite struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	UserID    string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_env_app_fav;index"`
	EnvID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_env_app_fav;index"`
	AppID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_env_app_fav;index"`

}
