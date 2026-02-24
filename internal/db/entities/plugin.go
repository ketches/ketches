package entities

type Plugin struct {
	Base
	ProjectID        string `gorm:"type:varchar(36);not null;uniqueIndex:idx_project_plugin_slug"`
	Slug             string `gorm:"type:varchar(64);not null;uniqueIndex:idx_project_plugin_slug"`
	Name             string `gorm:"type:varchar(128);not null"`
	Description      string `gorm:"type:text"`
	Image            string `gorm:"type:varchar(256);not null"`
	RegistryUsername string `gorm:"type:varchar(128)"`
	RegistryPassword string `gorm:"type:varchar(256)"`
	Command          string `gorm:"type:text"`
	EnvVars          string `gorm:"type:text"`
	PluginType       string `gorm:"type:varchar(16);not null"`
	InstallCount     int    `gorm:"->"`

	Project Project `gorm:"foreignKey:ProjectID"`
}
