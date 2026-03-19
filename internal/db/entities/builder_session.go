package entities

import "time"

type BuilderSessionStatus string

const (
	BuilderSessionStatusProvisioning BuilderSessionStatus = "provisioning"
	BuilderSessionStatusReady        BuilderSessionStatus = "ready"
	BuilderSessionStatusRunning      BuilderSessionStatus = "running"
	BuilderSessionStatusFailed       BuilderSessionStatus = "failed"
	BuilderSessionStatusArchived     BuilderSessionStatus = "archived"
	BuilderSessionStatusExpired      BuilderSessionStatus = "expired"
)

type BuilderSession struct {
	Base
	ProjectID      string               `gorm:"type:varchar(36);index;not null"`
	BuildEnvID     string               `gorm:"type:varchar(36);index;not null"`
	Title          string               `gorm:"type:varchar(255);not null"`
	Summary        string               `gorm:"type:text"`
	Status         BuilderSessionStatus `gorm:"type:varchar(32);not null;default:'provisioning';index"`
	CreatedBy      string               `gorm:"type:varchar(36);index;not null"`
	LastActivityAt time.Time            `gorm:"index"`
	ExpiresAt      *time.Time           `gorm:"index"`
}
