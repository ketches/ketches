package models

import "time"

type TriggerBuildRequest struct {
	GitRef     string `json:"git_ref"`
	ImageTag   string `json:"image_tag"`
	AutoDeploy *bool  `json:"auto_deploy"`
}

type BuildResponse struct {
	ID                          string       `json:"id"`
	CodeRepositoryID            string       `json:"code_repository_id,omitempty"`
	CodeRepositoryBuildConfigID string       `json:"code_repository_build_config_id,omitempty"`
	AppID                       string       `json:"app_id"`
	App                         *AppResponse `json:"app,omitempty"`
	BuildConfigID               string       `json:"build_config_id"`
	BuildNumber                 int          `json:"build_number"`
	Status                      string       `json:"status"`
	BuildEnvID                  string       `json:"build_env_id"`
	GitRepoURL                  string       `json:"git_repo_url"`
	GitRef                      string       `json:"git_ref"`
	GitCommitSHA                string       `json:"git_commit_sha"`
	GitCommitMsg                string       `json:"git_commit_msg"`
	ImageFullName               string       `json:"image_full_name"`
	TriggerType                 string       `json:"trigger_type"`
	TriggeredBy                 string       `json:"triggered_by"`
	JobName                     string       `json:"job_name"`
	JobNamespace                string       `json:"job_namespace"`
	StartedAt                   *time.Time   `json:"started_at"`
	CompletedAt                 *time.Time   `json:"completed_at"`
	Duration                    int          `json:"duration"`
	ErrorMessage                string       `json:"error_message"`
	CreatedAt                   time.Time    `json:"created_at"`
}

type DeployBuildRequest struct {
	// intentionally empty - deploy from this build's image
}

type RebuildRequest struct {
	ImageTag string `json:"image_tag"`
}
