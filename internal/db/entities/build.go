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

type BuildLogPersistStatus string

const (
	BuildLogPersistPending           BuildLogPersistStatus = "pending"
	BuildLogPersistSucceeded         BuildLogPersistStatus = "succeeded"
	BuildLogPersistFailed            BuildLogPersistStatus = "failed"
	BuildLogPersistSourceUnavailable BuildLogPersistStatus = "source_unavailable"
	BuildLogPersistExpired           BuildLogPersistStatus = "expired"
)

type Build struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// BuildSettingID: the config used for this build (required).
	BuildSettingID string `gorm:"type:varchar(36);not null;index"`

	BuildNumber int         `gorm:"type:int;not null"`
	Status      BuildStatus `gorm:"type:varchar(32);default:'pending';index:idx_builds_status"`

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

	LogPath          string                `gorm:"type:varchar(512)"`
	LogSize          int64                 `gorm:"type:bigint"`
	LogPersistStatus BuildLogPersistStatus `gorm:"type:varchar(32);default:'pending';index:idx_builds_log_persist_status"`
	LogPersistError  string                `gorm:"type:text"`
	LogPersistedAt   *time.Time
	LogExpireAt      *time.Time `gorm:"index:idx_builds_log_expire_at"`
}

func (Build) TableName() string {
	return "builds"
}
