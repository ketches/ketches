package models

import "time"

type TriggerBuildRequest struct {
	GitRef     string `json:"git_ref"`
	ImageTag   string `json:"image_tag"`
	AutoDeploy *bool  `json:"auto_deploy"`
}

type BuildResponse struct {
	ID               string `json:"id"`
	BuildSettingName string `json:"build_setting_name"`
	BuildSettingID   string `json:"build_setting_id"`

	App           *AppResponse `json:"app,omitempty"`
	BuildNumber   int          `json:"build_number"`
	Status        string       `json:"status"`
	BuildEnvID    string       `json:"build_env_id"`
	GitRepoURL    string       `json:"git_repo_url"`
	GitRef        string       `json:"git_ref"`
	GitCommitSHA  string       `json:"git_commit_sha"`
	GitCommitMsg  string       `json:"git_commit_msg"`
	ImageFullName string       `json:"image_full_name"`
	TriggerType   string       `json:"trigger_type"`
	TriggeredBy   string       `json:"triggered_by"`
	JobName       string       `json:"job_name"`
	JobNamespace  string       `json:"job_namespace"`
	StartedAt     *time.Time   `json:"started_at"`
	CompletedAt   *time.Time   `json:"completed_at"`
	Duration      int          `json:"duration"`
	ErrorMessage  string       `json:"error_message"`
	CreatedAt     time.Time    `json:"created_at"`
}

type DeploymentAppEnvSimpleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeploymentAppSimpleResponse struct {
	ID   string                          `json:"id"`
	Name string                          `json:"name"`
	Env  *DeploymentAppEnvSimpleResponse `json:"env,omitempty"`
}

type CodeRepositoryDeploymentResponse struct {
	ID             string                       `json:"id"`
	DeploymentID   string                       `json:"deployment_id"`
	BuildSettingID string                       `json:"build_setting_id"`
	BuildNumber    int                          `json:"build_number"`
	Status         string                       `json:"status"`
	GitRef         string                       `json:"git_ref"`
	ImageFullName  string                       `json:"image_full_name"`
	ErrorMessage   string                       `json:"error_message"`
	CreatedAt      time.Time                    `json:"created_at"`
	App            *DeploymentAppSimpleResponse `json:"app,omitempty"`
}

type DeployBuildRequest struct {
	// intentionally empty - deploy from this build's image
}

type RebuildRequest struct {
	ImageTag string `json:"image_tag"`
}
