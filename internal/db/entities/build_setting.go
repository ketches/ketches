package entities

import "time"

// BuildSetting stores build setting for either an App or a CodeRepository.
// Exactly one of AppID or CodeRepositoryID will be set.
type BuildSetting struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// Scope: exactly one of these is set.
	AppID            *string `gorm:"type:varchar(36);index"`
	CodeRepositoryID *string `gorm:"type:varchar(36);index"`

	// Display name — required for CodeRepository scope.
	Name string `gorm:"type:varchar(128)"`

	// Git source
	GitRef      string `gorm:"type:varchar(256);default:'main'"`
	GitUsername string `gorm:"type:varchar(128)"` // app scope only
	GitPassword string `gorm:"type:varchar(512)"` // app scope only

	// Build parameters
	DockerfilePath string `gorm:"type:varchar(256);default:'Dockerfile'"`
	BuildContext   string `gorm:"type:varchar(256);default:'.'"`
	BuildArgs      string `gorm:"type:text"`
	ImageName      string `gorm:"type:varchar(256);not null"`
	RegistryID     string `gorm:"type:varchar(36);not null;index"`

	// Trigger behavior
	AutoBuild  bool `gorm:"type:bool;default:false"`
	AutoDeploy bool `gorm:"type:bool;default:false"`

	// Webhook
	WebhookEnabled bool   `gorm:"type:bool;default:false"`
	WebhookSecret  string `gorm:"type:varchar(256)"`
}

func (BuildSetting) TableName() string { return "build_settings" }
