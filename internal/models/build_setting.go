package models

import "time"

type BuildArgPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CreateBuildSettingRequest struct {
	Name                 string         `json:"name" binding:"required"`
	GitRef               string         `json:"git_ref"`
	DockerfilePath       string         `json:"dockerfile_path"`
	BuildContext         string         `json:"build_context"`
	ImageName            string         `json:"image_name" binding:"required"`
	RegistryID           string         `json:"registry_id" binding:"required"`
	BuildArgs            string         `json:"build_args"`
	BuildArgPairs        []BuildArgPair `json:"build_arg_pairs"`
	Platforms            string         `json:"platforms"`
	RegistryCacheEnabled *bool          `json:"registry_cache_enabled"`
	RegistryCacheRef     string         `json:"registry_cache_ref"`
}

type UpdateRepoBuildSettingRequest struct {
	Name                 string         `json:"name"`
	GitRef               string         `json:"git_ref"`
	DockerfilePath       string         `json:"dockerfile_path"`
	BuildContext         string         `json:"build_context"`
	ImageName            string         `json:"image_name"`
	RegistryID           string         `json:"registry_id"`
	BuildArgs            string         `json:"build_args"`
	BuildArgPairs        []BuildArgPair `json:"build_arg_pairs"`
	Platforms            string         `json:"platforms"`
	RegistryCacheEnabled *bool          `json:"registry_cache_enabled"`
	RegistryCacheRef     string         `json:"registry_cache_ref"`
}

type BuildSettingResponse struct {
	ID                   string                   `json:"id"`
	CodeRepositoryID     string                   `json:"code_repository_id,omitempty"`
	Name                 string                   `json:"name"`
	GitRef               string                   `json:"git_ref"`
	GitUsername          string                   `json:"git_username,omitempty"`
	DockerfilePath       string                   `json:"dockerfile_path"`
	BuildContext         string                   `json:"build_context"`
	ImageName            string                   `json:"image_name"`
	RegistryID           string                   `json:"registry_id"`
	Registry             *RegistrySummaryResponse `json:"registry,omitempty"`
	BuildArgs            string                   `json:"build_args"`
	BuildArgPairs        []BuildArgPair           `json:"build_arg_pairs,omitempty"`
	Platforms            string                   `json:"platforms"`
	RegistryCacheEnabled bool                     `json:"registry_cache_enabled"`
	RegistryCacheRef     string                   `json:"registry_cache_ref"`
	CreatedAt            time.Time                `json:"created_at"`
	UpdatedAt            time.Time                `json:"updated_at"`
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
