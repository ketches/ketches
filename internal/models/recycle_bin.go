package models

import "time"

type RecycleBinAppResponse struct {
	ID             string    `json:"id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	EnvID          string    `json:"env_id"`
	EnvName        string    `json:"env_name"`
	ProjectID      string    `json:"project_id"`
	ProjectName    string    `json:"project_name"`
	ProjectSlug    string    `json:"project_slug"`
	AppType        string    `json:"app_type"`
	ContainerImage string    `json:"container_image"`
	DeletedAt      time.Time `json:"deleted_at"`
}

type ListRecycleBinAppResponse struct {
	Items      []RecycleBinAppResponse `json:"items"`
	Pagination PaginationResponse      `json:"pagination"`
}

type RecycleBinEnvResponse struct {
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	ProjectID        string    `json:"project_id"`
	ProjectName      string    `json:"project_name"`
	ProjectSlug      string    `json:"project_slug"`
	ClusterID        string    `json:"cluster_id"`
	ClusterName      string    `json:"cluster_name"`
	ClusterNamespace string    `json:"cluster_namespace"`
	DeletedAt        time.Time `json:"deleted_at"`
}

type ListRecycleBinEnvResponse struct {
	Items      []RecycleBinEnvResponse `json:"items"`
	Pagination PaginationResponse      `json:"pagination"`
}

type RecycleBinProjectResponse struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	DeletedAt   time.Time `json:"deleted_at"`
}

type ListRecycleBinProjectResponse struct {
	Items      []RecycleBinProjectResponse `json:"items"`
	Pagination PaginationResponse          `json:"pagination"`
}

type RestoreResourceRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type PermanentlyDeleteResourceRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type EnvDeletionConflictResponse struct {
	Apps []RecycleBinAppResponse `json:"apps"`
}
