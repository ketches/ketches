package models

import "time"

type BuildResponse struct {
	ID               string     `json:"id"`
	BuildSettingName string     `json:"build_setting_name"`
	BuildSettingID   string     `json:"build_setting_id"`
	BuildNumber      int        `json:"build_number"`
	Status           string     `json:"status"`
	BuildEnvID       string     `json:"build_env_id"`
	GitRepoURL       string     `json:"git_repo_url"`
	GitRef           string     `json:"git_ref"`
	GitCommitSHA     string     `json:"git_commit_sha"`
	GitCommitMsg     string     `json:"git_commit_msg"`
	ImageFullName    string     `json:"image_full_name"`
	TriggerType      string     `json:"trigger_type"`
	TriggeredBy      string     `json:"triggered_by"`
	JobName          string     `json:"job_name"`
	JobNamespace     string     `json:"job_namespace"`
	StartedAt        *time.Time `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	Duration         int        `json:"duration"`
	ErrorMessage     string     `json:"error_message"`
	CreatedAt        time.Time  `json:"created_at"`
}

type AppBuildResponse struct {
	ID                     string     `json:"id"`
	BuildSettingName       string     `json:"build_setting_name"`
	BuildSettingID         string     `json:"build_setting_id"`
	BuildNumber            int        `json:"build_number"`
	Status                 string     `json:"status"`
	DeployStatus           string     `json:"deploy_status" gorm:"column:deploy_status"`
	BuildEnvID             string     `json:"build_env_id"`
	GitRepoURL             string     `json:"git_repo_url"`
	GitRef                 string     `json:"git_ref"`
	GitCommitSHA           string     `json:"git_commit_sha"`
	GitCommitMsg           string     `json:"git_commit_msg"`
	ImageFullName          string     `json:"image_full_name"`
	TriggerType            string     `json:"trigger_type"`
	TriggeredBy            string     `json:"triggered_by"`
	JobName                string     `json:"job_name"`
	JobNamespace           string     `json:"job_namespace"`
	StartedAt              *time.Time `json:"started_at"`
	CompletedAt            *time.Time `json:"completed_at"`
	Duration               int        `json:"duration"`
	ErrorMessage           string     `json:"error_message"`
	DeploymentErrorMessage string     `json:"deployment_error_message" gorm:"column:deployment_error_message"`
	CreatedAt              time.Time  `json:"created_at"`
}

type CodeRepositoryDeploymentResponse struct {
	ID             string    `json:"id"`
	DeploymentID   string    `json:"deployment_id"`
	BuildSettingID string    `json:"build_setting_id"`
	BuildNumber    int       `json:"build_number"`
	Status         string    `json:"status"`
	GitRef         string    `json:"git_ref"`
	ImageFullName  string    `json:"image_full_name"`
	ErrorMessage   string    `json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
	AppID          string    `json:"app_id"`
	AppName        string    `json:"app_name"`
	EnvName        string    `json:"env_name"`
}
