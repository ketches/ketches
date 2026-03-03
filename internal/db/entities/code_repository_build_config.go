package entities

import "time"

// CodeRepositoryBuildConfig represents one build configuration under a code repository.
// A repo can have multiple configs (e.g. multi-project repo: frontend, backend).
type CodeRepositoryBuildConfig struct {
	ID               string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
	CodeRepositoryID string    `gorm:"type:varchar(36);index;not null"`

	Name   string `gorm:"type:varchar(128);not null"`
	GitRef string `gorm:"type:varchar(256);default:'main'"`

	DockerfilePath string `gorm:"type:varchar(256);default:'Dockerfile'"`
	BuildContext   string `gorm:"type:varchar(256);default:'.'"`
	ImageName      string `gorm:"type:varchar(256);not null"`
	RegistryID     string `gorm:"type:varchar(36);not null"`
	BuildArgs      string `gorm:"type:text"`

	AutoBuild      bool `gorm:"type:bool;default:false"`
	AutoDeploy     bool `gorm:"type:bool;default:false"`
	WebhookEnabled bool `gorm:"type:bool;default:false"` // when true, repo webhook triggers this config

	CodeRepository CodeRepository    `gorm:"foreignKey:CodeRepositoryID"`
	Registry       ContainerRegistry `gorm:"foreignKey:RegistryID"`
}

func (CodeRepositoryBuildConfig) TableName() string {
	return "code_repository_build_configs"
}
