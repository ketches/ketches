package models

import "time"

type SimpleCodeRepository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CreateCodeRepositoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	GitRepoURL  string `json:"git_repo_url" binding:"required"`
	GitUsername string `json:"git_username"`
	GitPassword string `json:"git_password"`
}

type UpdateCodeRepositoryRequest struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	GitRepoURL       string `json:"git_repo_url"`
	GitUsername      string `json:"git_username"`
	GitPassword      string `json:"git_password"`
	ClearGitPassword *bool  `json:"clear_git_password,omitempty"`
}

type CodeRepositoryResponse struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	GitRepoURL     string    `json:"git_repo_url"`
	GitUsername    string    `json:"git_username"`
	GitPassword    string    `json:"-"`
	HasGitPassword bool      `json:"has_git_password"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ListCodeRepositoryResponse struct {
	Items      []CodeRepositoryResponse `json:"items"`
	Pagination PaginationResponse       `json:"pagination"`
}

type TriggerCodeRepositoryBuildRequest struct {
	BuildSettingID string `json:"build_setting_id" binding:"required"`
	BuildEnvID     string `json:"build_env_id" binding:"required"`
	GitRef         string `json:"git_ref"`
	ImageTag       string `json:"image_tag"`
	AutoDeploy     *bool  `json:"auto_deploy"`

	DeployEnvID   string `json:"deploy_env_id"`
	DeployAppID   string `json:"deploy_app_id"`
	DeployAppName string `json:"deploy_app_name"`
	DeployAppSlug string `json:"deploy_app_slug"`
}

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
