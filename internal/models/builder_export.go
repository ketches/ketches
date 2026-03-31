package models

import "time"

type BuilderExportResponse struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	RunID        string    `json:"run_id"`
	WorkspaceID  string    `json:"workspace_id"`
	SnapshotID   string    `json:"snapshot_id"`
	Kind         string    `json:"kind"`
	Status       string    `json:"status"`
	FileName     string    `json:"file_name"`
	StoragePath  string    `json:"storage_path"`
	SourceRoot   string    `json:"source_root"`
	FileCount    int       `json:"file_count"`
	SizeBytes    int64     `json:"size_bytes"`
	MetadataJSON string    `json:"metadata_json"`
	ErrorMessage string    `json:"error_message"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PromoteBuilderExportToCodeRepositoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	GitRepoURL  string `json:"git_repo_url" binding:"required"`
	GitUsername string `json:"git_username"`
	GitPassword string `json:"git_password"`
}

type BuilderExportPromotionResponse struct {
	Export     BuilderExportResponse  `json:"export"`
	Repository CodeRepositoryResponse `json:"repository"`
}

type BuilderExportPromotionPlanResponse struct {
	Export                    BuilderExportResponse `json:"export"`
	SourceKind                string                `json:"source_kind"`
	PlannedProjectKind        string                `json:"planned_project_kind"`
	SuggestedRepositoryName   string                `json:"suggested_repository_name"`
	SuggestedRepositorySlug   string                `json:"suggested_repository_slug"`
	SuggestedBuildEnvID       string                `json:"suggested_build_env_id"`
	SuggestedBuildSettingName string                `json:"suggested_build_setting_name"`
	SuggestedImageName        string                `json:"suggested_image_name"`
	SuggestedDockerfilePath   string                `json:"suggested_dockerfile_path"`
	SuggestedBuildContext     string                `json:"suggested_build_context"`
	CanTriggerInitialBuild    bool                  `json:"can_trigger_initial_build"`
	RequiresRegistrySelection bool                  `json:"requires_registry_selection"`
	MissingRequirements       []string              `json:"missing_requirements"`
}

type PromoteBuilderExportToInitialBuildRequest struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	GitRepoURL       string `json:"git_repo_url" binding:"required"`
	GitUsername      string `json:"git_username"`
	GitPassword      string `json:"git_password"`
	BuildEnvID       string `json:"build_env_id" binding:"required"`
	RegistryID       string `json:"registry_id" binding:"required"`
	BuildSettingName string `json:"build_setting_name"`
	ImageName        string `json:"image_name"`
	DockerfilePath   string `json:"dockerfile_path"`
	BuildContext     string `json:"build_context"`
	GitRef           string `json:"git_ref"`
}

type BuilderExportInitialBuildPromotionResponse struct {
	Promotion    BuilderExportPromotionResponse `json:"promotion"`
	BuildSetting BuildSettingResponse           `json:"build_setting"`
	Build        BuildResponse                  `json:"build"`
}

type DeployBuilderExportBuildRequest struct {
	RepositoryID string `json:"repository_id" binding:"required"`
	BuildID      string `json:"build_id" binding:"required"`
	TargetEnvID  string `json:"target_env_id" binding:"required"`
	AppID        string `json:"app_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
}
