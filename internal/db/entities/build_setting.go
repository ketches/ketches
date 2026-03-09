package entities

import "time"

// BuildSetting stores build setting for either an App or a CodeRepository.
// Exactly one of AppID or CodeRepositoryID will be set.
type BuildSetting struct {
	ID               string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
	Name             string    `gorm:"type:varchar(128)"`
	AppID            *string   `gorm:"type:varchar(36);index"`
	CodeRepositoryID *string   `gorm:"type:varchar(36);index"`

	// Git source
	GitRef string `gorm:"type:varchar(256);default:'main'"`

	// Build parameters
	DockerfilePath string `gorm:"type:varchar(256);default:'Dockerfile'"`
	BuildContext   string `gorm:"type:varchar(256);default:'.'"`
	BuildArgs      string `gorm:"type:text"`
	ImageName      string `gorm:"type:varchar(256);not null"`
	RegistryID     string `gorm:"type:varchar(36);not null;index"`
}

func (BuildSetting) TableName() string { return "build_settings" }
