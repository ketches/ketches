package models

import "time"

type UpsertAppBuildSettingRequest struct {
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

type CreateRepoBuildSettingRequest struct {
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

type UpdateRepoBuildSettingRequest struct {
	Name           string `json:"name"`
	GitRef         string `json:"git_ref"`
	DockerfilePath string `json:"dockerfile_path"`
	BuildContext   string `json:"build_context"`
	ImageName      string `json:"image_name"`
	RegistryID     string `json:"registry_id"`
	BuildArgs      string `json:"build_args"`
	AutoBuild      bool   `json:"auto_build"`
	AutoDeploy     bool   `json:"auto_deploy"`
	WebhookEnabled bool   `json:"webhook_enabled"`
}

type BuildSettingResponse struct {
	ID               string                     `json:"id"`
	AppID            string                     `json:"app_id,omitempty"`
	CodeRepositoryID string                     `json:"code_repository_id,omitempty"`
	Name             string                     `json:"name"`
	GitRef           string                     `json:"git_ref"`
	GitUsername      string                     `json:"git_username,omitempty"`
	DockerfilePath   string                     `json:"dockerfile_path"`
	BuildContext     string                     `json:"build_context"`
	ImageName        string                     `json:"image_name"`
	RegistryID       string                     `json:"registry_id"`
	Registry         *RegistrySummaryResponse   `json:"registry,omitempty"`
	BuildArgs        string                     `json:"build_args"`
	AutoBuild        bool                       `json:"auto_build"`
	AutoDeploy       bool                       `json:"auto_deploy"`
	WebhookEnabled   bool                       `json:"webhook_enabled"`
	WebhookSecret    string                     `json:"webhook_secret,omitempty"`
	WebhookURL       string                     `json:"webhook_url,omitempty"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
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
