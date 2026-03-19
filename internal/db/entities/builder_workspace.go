package entities

import "time"

type BuilderWorkspaceStatus string

const (
	BuilderWorkspaceStatusProvisioning BuilderWorkspaceStatus = "provisioning"
	BuilderWorkspaceStatusActive       BuilderWorkspaceStatus = "active"
	BuilderWorkspaceStatusFailed       BuilderWorkspaceStatus = "failed"
	BuilderWorkspaceStatusExpired      BuilderWorkspaceStatus = "expired"
)

type BuilderWorkspace struct {
	ID            string                 `gorm:"type:varchar(36);primaryKey"`
	CreatedAt     time.Time              `gorm:"autoCreateTime"`
	UpdatedAt     time.Time              `gorm:"autoUpdateTime"`
	SessionID     string                 `gorm:"type:varchar(36);index;not null"`
	BuildEnvID    string                 `gorm:"type:varchar(36);index;not null"`
	ClusterID     string                 `gorm:"type:varchar(36);index"`
	Namespace     string                 `gorm:"type:varchar(255);index"`
	PodName       string                 `gorm:"type:varchar(255);index"`
	ContainerName string                 `gorm:"type:varchar(255)"`
	Status        BuilderWorkspaceStatus `gorm:"type:varchar(32);not null;default:'provisioning';index"`
	WorkspaceRoot string                 `gorm:"type:varchar(512)"`
	TerminatedAt  *time.Time             `gorm:"index"`
}
