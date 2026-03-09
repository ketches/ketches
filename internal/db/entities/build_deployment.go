package entities

import "time"

// BuildDeploymentStatus represents the state of a single deploy event.
type BuildDeploymentStatus string

const (
	BuildDeploymentStatusPending  BuildDeploymentStatus = "pending"
	BuildDeploymentStatusDeployed BuildDeploymentStatus = "deployed"
	BuildDeploymentStatusFailed   BuildDeploymentStatus = "failed"
)

// BuildDeployment records a single deployment event: one build deployed to one app.
// A build may have multiple deployment records (1:N).
type BuildDeployment struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// BuildID references the build that was deployed.
	BuildID string `gorm:"type:varchar(36);not null;index"`

	// AppID is the target app. Null when the build was triggered for a new app not yet created.
	AppID *string `gorm:"type:varchar(36);index"`
	// EnvID is the target environment.
	EnvID string `gorm:"type:varchar(36);not null"`

	// AppName and AppSlug are only used when creating a brand-new app during deployment.
	AppName string `gorm:"type:varchar(128)"`
	AppSlug string `gorm:"type:varchar(64)"`

	// Status tracks whether the deployment is pending, deployed, or failed.
	Status BuildDeploymentStatus `gorm:"type:varchar(32);default:'pending'"`

	// DeployedBy is the user ID who triggered the deploy, or "auto".
	DeployedBy string `gorm:"type:varchar(64)"`

	DeployedAt   *time.Time
	ErrorMessage string `gorm:"type:text"`
}

func (BuildDeployment) TableName() string {
	return "build_deployments"
}
