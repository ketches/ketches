package models

import (
	"time"

	"github.com/ketches/ketches/internal/db/entities"
)

type CreateEnvRequest struct {
	Slug             string `json:"slug" binding:"required"`
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	ProjectID        string `json:"project_id"`
	ClusterID        string `json:"cluster_id" binding:"required"`
	ClusterNamespace string `json:"cluster_namespace" binding:"required"`
	IsBuildEnv       bool   `json:"is_build_env"`
}

type CheckEnvNamespaceAvailabilityRequest struct {
	ClusterID        string `form:"cluster_id" json:"cluster_id" binding:"required"`
	ClusterNamespace string `form:"cluster_namespace" json:"cluster_namespace" binding:"required"`
}

type EnvNamespaceAvailabilityResponse struct {
	Available bool   `json:"available"`
	Source    string `json:"source"`
	Message   string `json:"message"`
}

type EnvResponse struct {
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	ProjectID        string    `json:"project_id"`
	ClusterID        string    `json:"cluster_id"`
	ClusterName      string    `json:"cluster_name"`
	ProjectName      string    `json:"project_name"`
	ClusterNamespace string    `json:"cluster_namespace"`
	IsBuildEnv       bool      `json:"is_build_env"`
	CreatedAt        time.Time `json:"created_at"`
}

type ListEnvResponse struct {
	Items      []EnvResponse      `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

// RecycleBinEnvRow represents a flattened environment record for the recycle bin list
type RecycleBinEnvRow struct {
	entities.Env
	ProjectName string `gorm:"column:project_name"`
	ProjectSlug string `gorm:"column:project_slug"`
	ClusterName string `gorm:"column:cluster_name"`
}

type ResourceQuotaResponse struct {
	CPURequest    string `json:"cpu_request"`
	CPULimit      string `json:"cpu_limit"`
	MemoryRequest string `json:"memory_request"`
	MemoryLimit   string `json:"memory_limit"`
	Pods          string `json:"pods"`
}

type UpdateResourceQuotaRequest struct {
	CPURequest    string `json:"cpu_request" binding:"required"`
	CPULimit      string `json:"cpu_limit" binding:"required"`
	MemoryRequest string `json:"memory_request" binding:"required"`
	MemoryLimit   string `json:"memory_limit" binding:"required"`
	Pods          string `json:"pods" binding:"required"`
}
