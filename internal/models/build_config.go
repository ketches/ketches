package models

import "time"

type UpsertBuildConfigRequest struct {
	GitRepoURL     string `json:"git_repo_url" binding:"required"`
	GitRef         string `json:"git_ref"`
	GitUsername    string `json:"git_username"`
	GitPassword    string `json:"git_password"`
	DockerfilePath string `json:"dockerfile_path"`
	BuildContext   string `json:"build_context"`
	ImageName      string `json:"image_name" binding:"required"`
	RegistryID     string `json:"registry_id" binding:"required"`
	BuildArgs      string `json:"build_args"`
	AutoBuild      bool   `json:"auto_build"`
	AutoDeploy     bool   `json:"auto_deploy"`
	WebhookEnabled bool   `json:"webhook_enabled"`
}

type BuildConfigResponse struct {
	ID             string                 `json:"id"`
	AppID          string                 `json:"app_id"`
	GitRepoURL     string                 `json:"git_repo_url"`
	GitRef         string                 `json:"git_ref"`
	GitUsername    string                 `json:"git_username"`
	DockerfilePath string                 `json:"dockerfile_path"`
	BuildContext   string                 `json:"build_context"`
	ImageName      string                 `json:"image_name"`
	RegistryID     string                 `json:"registry_id"`
	Registry       *ContainerRegistryResponse `json:"registry,omitempty"`
	BuildArgs      string                 `json:"build_args"`
	AutoBuild      bool                   `json:"auto_build"`
	AutoDeploy     bool                   `json:"auto_deploy"`
	WebhookSecret  string                 `json:"webhook_secret"`
	WebhookEnabled bool                   `json:"webhook_enabled"`
	WebhookURL     string                 `json:"webhook_url"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type TestGitConnectionRequest struct {
	GitRepoURL  string `json:"git_repo_url" binding:"required"`
	GitRef      string `json:"git_ref"`
	GitUsername string `json:"git_username"`
	GitPassword string `json:"git_password"`
}

type TestGitConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
