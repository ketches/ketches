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
	BuildTriggerManual  BuildTriggerType = "manual"
	BuildTriggerWebhook BuildTriggerType = "webhook"
	BuildTriggerAuto    BuildTriggerType = "auto"
)

type Build struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	// CodeRepositoryID: build belongs to this code repo. Optional for backward compat with app-bound builds.
	CodeRepositoryID *string `gorm:"type:varchar(36);index"`
	// CodeRepositoryBuildConfigID: which repo build config was used (required when CodeRepositoryID is set).
	CodeRepositoryBuildConfigID *string `gorm:"type:varchar(36);index"`
	// AppID: optional deployment record only — "this build was deployed to this app". Build produces an image; deploy (to any env/app) is separate. No FK to apps.
	AppID         *string     `gorm:"type:varchar(36);index"`
	BuildConfigID *string     `gorm:"type:varchar(36);index"` // optional: app build config (legacy); NULL for code-repo builds
	BuildNumber   int         `gorm:"type:int;not null"`
	Status        BuildStatus `gorm:"type:varchar(32);default:'pending'"`

	// Build environment (which env's cluster/namespace ran this build)
	BuildEnvID string `gorm:"type:varchar(36);not null"`

	// Git info (snapshot at build time)
	GitRepoURL   string `gorm:"type:varchar(512)"`
	GitRef       string `gorm:"type:varchar(256)"`
	GitCommitSHA string `gorm:"type:varchar(64)"`
	GitCommitMsg string `gorm:"type:text"`

	// Image output
	ImageFullName string `gorm:"type:varchar(512)"`

	// Execution info
	TriggerType  BuildTriggerType `gorm:"type:varchar(32);not null"`
	TriggeredBy  string           `gorm:"type:varchar(36)"`
	JobName      string           `gorm:"type:varchar(256)"`
	JobNamespace string           `gorm:"type:varchar(256)"`
	StartedAt    *time.Time
	CompletedAt  *time.Time
	Duration     int    `gorm:"type:int"`
	ErrorMessage string `gorm:"type:text"`

	PendingDeployEnvID   string `gorm:"type:varchar(36)"`
	PendingDeployAppID   string `gorm:"type:varchar(36)"`
	PendingDeployAppName string `gorm:"type:varchar(128)"`
	PendingDeployAppSlug string `gorm:"type:varchar(64)"`

	// Relationships (no FK to App: build only produces image; deploy to app is a separate action)
	CodeRepository      *CodeRepository            `gorm:"foreignKey:CodeRepositoryID"`
	CodeRepoBuildConfig *CodeRepositoryBuildConfig `gorm:"foreignKey:CodeRepositoryBuildConfigID"`
	App                 *App                       `gorm:"foreignKey:AppID;references:ID;constraint:false"`         // optional, no DB constraint
	BuildConfig         *AppBuildConfig            `gorm:"foreignKey:BuildConfigID;references:ID;constraint:false"` // optional, no DB constraint
	BuildEnv            Env                        `gorm:"foreignKey:BuildEnvID"`
}

func (Build) TableName() string {
	return "builds"
}
