package entities

// AppGroup represents a named group of apps within a project.
type AppGroup struct {
	Base
	ProjectID       string `gorm:"type:varchar(36);not null;index"`
	Name            string `gorm:"type:varchar(128);not null"`
	Description     string `gorm:"type:text"`
	CreatedByUserID string `gorm:"type:varchar(36);not null"`
}

// AppGroupMember maps apps to groups (many-to-many).
type AppGroupMember struct {
	Base
	GroupID string `gorm:"type:varchar(36);not null;uniqueIndex:idx_group_app"`
	AppID   string `gorm:"type:varchar(36);not null;uniqueIndex:idx_group_app"`
}

// AppFavorite records a user's favorite app.
type AppFavorite struct {
	Base
	UserID string `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_app_fav"`
	AppID  string `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_app_fav"`
}
