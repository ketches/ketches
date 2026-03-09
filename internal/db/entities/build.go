package entities

import "time"

type BuildStatus string

const (
	BuildStatusPending   BuildStatus = "pending"
	BuildStatusCloning   BuildStatus = "cloning"
	BuildStatusBuilding  BuildStatus = "building"
	BuildStatusSucceeded BuildStatus = "succeeded"
	BuildStatusFailed    BuildStatus = "failed"
	BuildStatusCancelled BuildStatus = "cancelled"
)

type BuildTriggerType string

const (
	BuildTriggerManual BuildTriggerType = "manual"
	BuildTriggerAuto   BuildTriggerType = "auto"
)

type Build struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// BuildSettingID: the config used for this build (required).
	BuildSettingID string `gorm:"type:varchar(36);not null;index"`

	BuildNumber int         `gorm:"type:int;not null"`
	Status      BuildStatus `gorm:"type:varchar(32);default:'pending'"`

	// Build environment
	BuildEnvID string `gorm:"type:varchar(36);not null;index"`

	// Git info (snapshot at build time)
	GitRepoURL   string `gorm:"type:varchar(512)"`
	GitRef       string `gorm:"type:varchar(256)"`
	GitCommitSHA string `gorm:"type:varchar(64)"`
	GitCommitMsg string `gorm:"type:text"`

	// Build artifact
	ImageFullName string `gorm:"type:varchar(512)"`

	// Execution info
	TriggerType  BuildTriggerType `gorm:"type:varchar(32);not null"`
	TriggeredBy  *string          `gorm:"type:varchar(36)"`
	JobName      string           `gorm:"type:varchar(256)"`
	JobNamespace string           `gorm:"type:varchar(256)"`
	StartedAt    *time.Time
	CompletedAt  *time.Time
	Duration     int    `gorm:"type:int"`
	ErrorMessage string `gorm:"type:text"`
}

func (Build) TableName() string {
	return "builds"
}
