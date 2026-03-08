package models

import "time"

type CreateContainerRegistryRequest struct {
	Name          string `json:"name" binding:"required"`
	Provider      string `json:"provider" binding:"required,oneof=dockerhub harbor ghcr acr ecr aliyun custom"`
	Endpoint      string `json:"endpoint" binding:"required"`
	SkipTLSVerify *bool  `json:"skip_tls_verify"`
	Namespace     string `json:"namespace"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	IsDefault     bool   `json:"is_default"`
	Enabled       bool   `json:"enabled"`
	Description   string `json:"description"`
}

type UpdateContainerRegistryRequest struct {
	Name          string `json:"name"`
	Provider      string `json:"provider" binding:"omitempty,oneof=dockerhub harbor ghcr acr ecr aliyun custom"`
	Endpoint      string `json:"endpoint"`
	SkipTLSVerify *bool  `json:"skip_tls_verify"`
	Namespace     string `json:"namespace"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	IsDefault     *bool  `json:"is_default"`
	Enabled       *bool  `json:"enabled"`
	Description   string `json:"description"`
}

type ContainerRegistryResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Provider      string    `json:"provider"`
	Endpoint      string    `json:"endpoint"`
	SkipTLSVerify bool      `json:"skip_tls_verify"`
	Namespace     string    `json:"namespace"`
	Username      string    `json:"username"`
	Password      string    `json:"password,omitempty"`
	Scope         string    `json:"scope"`
	ClusterID     string    `json:"cluster_id,omitempty"`
	ProjectID     string    `json:"project_id,omitempty"`
	IsDefault     bool      `json:"is_default"`
	Enabled       bool      `json:"enabled"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

// RegistrySummaryResponse is a minimal registry view embedded in build-setting responses.
// Only the fields actually rendered by the frontend are included.
type RegistrySummaryResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type ListContainerRegistryResponse struct {
	Items      []ContainerRegistryResponse `json:"items"`
	Pagination PaginationResponse          `json:"pagination"`
}

type TestContainerRegistryRequest struct {
	Provider      string `json:"provider" binding:"required,oneof=dockerhub harbor ghcr acr ecr custom"`
	Endpoint      string `json:"endpoint" binding:"required"`
	SkipTLSVerify bool   `json:"skip_tls_verify"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type TestContainerRegistryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
