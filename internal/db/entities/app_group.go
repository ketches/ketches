package entities

import "time"

// AppGroup represents a named group of apps within an environment.
type AppGroup struct {
	ID          string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	EnvID       string    `gorm:"type:varchar(36);not null;index"`
	Name        string    `gorm:"type:varchar(128);not null"`
	Description string    `gorm:"type:text"`
}

// AppGroupMember maps apps to groups (many-to-many).
type AppGroupMember struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	GroupID   string    `gorm:"type:varchar(36);not null;index:idx_app_group_members_group_id;uniqueIndex:idx_group_app"`
	AppID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_group_app"`
}
