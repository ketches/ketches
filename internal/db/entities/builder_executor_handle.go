package entities

import "time"

type BuilderExecutorHandleKind string

const (
	BuilderExecutorHandleKindWorkspacePod BuilderExecutorHandleKind = "workspace_pod"
	BuilderExecutorHandleKindBuildJob     BuilderExecutorHandleKind = "build_job"
	BuilderExecutorHandleKindSandbox      BuilderExecutorHandleKind = "sandbox_session"
)

type BuilderExecutorHandleStatus string

const (
	BuilderExecutorHandleStatusProvisioning BuilderExecutorHandleStatus = "provisioning"
	BuilderExecutorHandleStatusActive       BuilderExecutorHandleStatus = "active"
	BuilderExecutorHandleStatusTerminating  BuilderExecutorHandleStatus = "terminating"
	BuilderExecutorHandleStatusTerminated   BuilderExecutorHandleStatus = "terminated"
	BuilderExecutorHandleStatusFailed       BuilderExecutorHandleStatus = "failed"
)

type BuilderExecutorHandle struct {
	ID              string                      `gorm:"type:varchar(36);primaryKey"`
	CreatedAt       time.Time                   `gorm:"autoCreateTime"`
	UpdatedAt       time.Time                   `gorm:"autoUpdateTime"`
	SessionID       string                      `gorm:"type:varchar(36);index;not null"`
	RunID           *string                     `gorm:"type:varchar(36);index"`
	Kind            BuilderExecutorHandleKind   `gorm:"type:varchar(32);not null;index"`
	Status          BuilderExecutorHandleStatus `gorm:"type:varchar(32);not null;default:'provisioning';index"`
	ClusterID       *string                     `gorm:"type:varchar(36);index"`
	Namespace       *string                     `gorm:"type:varchar(255);index"`
	WorkloadName    *string                     `gorm:"type:varchar(255);index"`
	ContainerName   *string                     `gorm:"type:varchar(255)"`
	ExternalRef     *string                     `gorm:"type:varchar(512);index"`
	LeaseToken      *string                     `gorm:"type:varchar(64);index"`
	LeaseExpiresAt  *time.Time                  `gorm:"index"`
	LastHeartbeatAt *time.Time                  `gorm:"index"`
	TerminatedAt    *time.Time                  `gorm:"index"`
	MetadataJSON    string                      `gorm:"type:text"`
}
