package entities

import "time"

type AppBuildConfig struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	AppID     string    `gorm:"type:varchar(36);uniqueIndex;not null"`

	// Git source
	GitRepoURL  string `gorm:"type:varchar(512);not null"`
	GitRef      string `gorm:"type:varchar(256);default:'main'"`
	GitUsername string `gorm:"type:varchar(128)"`
	GitPassword string `gorm:"type:varchar(512)"`

	// Build settings
	DockerfilePath string `gorm:"type:varchar(256);default:'Dockerfile'"`
	BuildContext   string `gorm:"type:varchar(256);default:'.'"`
	ImageName      string `gorm:"type:varchar(256);not null"`
	RegistryID     string `gorm:"type:varchar(36);not null;index"`

	// Build behavior
	BuildArgs  string `gorm:"type:text"`
	AutoBuild  bool   `gorm:"type:bool;default:false"`
	AutoDeploy bool   `gorm:"type:bool;default:false"`

	// Webhook
	WebhookSecret  string `gorm:"type:varchar(256)"`
	WebhookEnabled bool   `gorm:"type:bool;default:false"`
}

func (AppBuildConfig) TableName() string {
	return "app_build_configs"
}
