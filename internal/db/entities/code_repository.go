package entities

// CodeRepository represents a Git repository managed in the system.
// Step 1: only repo identity and credentials. Name defaults to repo name from URL, editable.
// Build configs (branch, Dockerfile, image, registry, etc.) are managed per-repo in CodeRepositoryBuildConfig.
type CodeRepository struct {
	Base
	ProjectID string `gorm:"type:varchar(36);not null;uniqueIndex:idx_project_code_repo_slug;index"`

	// Display name (default from URL repo name, editable)
	Name string `gorm:"type:varchar(128);not null"`
	// URL-friendly identifier (default from name, editable)
	Slug string `gorm:"type:varchar(128);not null;default:'';uniqueIndex:idx_project_code_repo_slug"`

	// Git source
	GitRepoURL  string `gorm:"type:varchar(512);not null"`
	GitUsername string `gorm:"type:varchar(128)"`
	GitPassword string `gorm:"type:varchar(512)"`

	// Webhook (one per repo; which build configs to trigger is per BuildConfig.WebhookEnabled)
	WebhookSecret  string `gorm:"type:varchar(256)"`
	WebhookEnabled bool   `gorm:"type:bool;default:false"`

	Project      Project                     `gorm:"foreignKey:ProjectID"`
	BuildConfigs []CodeRepositoryBuildConfig `gorm:"foreignKey:CodeRepositoryID"`
	Builds       []Build                     `gorm:"foreignKey:CodeRepositoryID"`
}

func (CodeRepository) TableName() string {
	return "code_repositories"
}
