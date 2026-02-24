package models

import "time"

// CreateCodeRepositoryRequest: step 1 - only repo URL and credentials. Name/slug optional (derived from URL).
type CreateCodeRepositoryRequest struct {
	Name        string `json:"name"` // optional; derived from git_repo_url if empty
	Slug        string `json:"slug"` // optional; derived from name if empty
	GitRepoURL  string `json:"git_repo_url" binding:"required"`
	GitUsername string `json:"git_username"`
	GitPassword string `json:"git_password"`
}

// UpdateCodeRepositoryRequest: name, slug, url, credentials, webhook only.
type UpdateCodeRepositoryRequest struct {
	Name           string `json:"name"`
	GitRepoURL     string `json:"git_repo_url"`
	GitUsername    string `json:"git_username"`
	GitPassword    string `json:"git_password"`
	WebhookEnabled *bool  `json:"webhook_enabled"`
}

// CodeRepositoryResponse: repo identity and webhook; no build fields.
type CodeRepositoryResponse struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	GitRepoURL     string    `json:"git_repo_url"`
	GitUsername    string    `json:"git_username"`
	GitPassword    string    `json:"git_password"`
	WebhookSecret  string    `json:"webhook_secret"`
	WebhookEnabled bool      `json:"webhook_enabled"`
	WebhookURL     string    `json:"webhook_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateCodeRepositoryBuildConfigRequest: used when adding a build config under a repo.
type CreateCodeRepositoryBuildConfigRequest struct {
	Name           string `json:"name" binding:"required"`
	GitRef         string `json:"git_ref"`
	DockerfilePath string `json:"dockerfile_path"`
	BuildContext   string `json:"build_context"`
	ImageName      string `json:"image_name" binding:"required"`
	RegistryID     string `json:"registry_id" binding:"required"`
	BuildArgs      string `json:"build_args"`
	AutoBuild      bool   `json:"auto_build"`
	AutoDeploy     bool   `json:"auto_deploy"`
	WebhookEnabled bool   `json:"webhook_enabled"`
}

// UpdateCodeRepositoryBuildConfigRequest
type UpdateCodeRepositoryBuildConfigRequest struct {
	Name           string `json:"name"`
	GitRef         string `json:"git_ref"`
	DockerfilePath string `json:"dockerfile_path"`
	BuildContext   string `json:"build_context"`
	ImageName      string `json:"image_name"`
	RegistryID     string `json:"registry_id"`
	BuildArgs      string `json:"build_args"`
	AutoBuild      *bool  `json:"auto_build"`
	AutoDeploy     *bool  `json:"auto_deploy"`
	WebhookEnabled *bool  `json:"webhook_enabled"`
}

// CodeRepositoryBuildConfigResponse
type CodeRepositoryBuildConfigResponse struct {
	ID               string                     `json:"id"`
	CodeRepositoryID string                     `json:"code_repository_id"`
	Name             string                     `json:"name"`
	GitRef           string                     `json:"git_ref"`
	DockerfilePath   string                     `json:"dockerfile_path"`
	BuildContext     string                     `json:"build_context"`
	ImageName        string                     `json:"image_name"`
	RegistryID       string                     `json:"registry_id"`
	Registry         *ContainerRegistryResponse `json:"registry,omitempty"`
	BuildArgs        string                     `json:"build_args"`
	AutoBuild        bool                       `json:"auto_build"`
	AutoDeploy       bool                       `json:"auto_deploy"`
	WebhookEnabled   bool                       `json:"webhook_enabled"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

// TriggerCodeRepositoryBuildRequest: build_config_id required; build runs in build_env_id.
type TriggerCodeRepositoryBuildRequest struct {
	BuildConfigID string `json:"build_config_id" binding:"required"`
	BuildEnvID    string `json:"build_env_id" binding:"required"`
	GitRef        string `json:"git_ref"`
	ImageTag      string `json:"image_tag"`
	AutoDeploy    *bool  `json:"auto_deploy"`

	DeployEnvID   string `json:"deploy_env_id"`
	DeployAppID   string `json:"deploy_app_id"`
	DeployAppName string `json:"deploy_app_name"`
	DeployAppSlug string `json:"deploy_app_slug"`
}

// DeployCodeRepositoryBuildRequest unchanged
type DeployCodeRepositoryBuildRequest struct {
	TargetEnvID string `json:"target_env_id" binding:"required"`
	AppID       string `json:"app_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
}

type GitRef struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ListGitRefsResponse struct {
	Refs []GitRef `json:"refs"`
}
