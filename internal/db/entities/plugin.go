package entities

import "time"

type Plugin struct {
	ID               string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
	ProjectID        string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_project_plugin_slug;index"`
	Slug             string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_project_plugin_slug"`
	Name             string    `gorm:"type:varchar(128);not null"`
	Description      string    `gorm:"type:text"`
	Image            string    `gorm:"type:varchar(256);not null"`
	ImagePullPolicy  string    `gorm:"type:varchar(32);default:'IfNotPresent'"`
	RegistryUsername string    `gorm:"type:varchar(128)"`
	RegistryPassword string    `gorm:"type:varchar(256)"`
	Command          string    `gorm:"type:text"`
	EnvVars          string    `gorm:"type:text"`
	PluginType       string    `gorm:"type:varchar(16);not null"`
	InstallCount     int       `gorm:"->"`
}
