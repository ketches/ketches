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

type EnvResponse struct {
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	ProjectID        string    `json:"project_id"`
	ClusterID        string    `json:"cluster_id"`
	ClusterNamespace string    `json:"cluster_namespace"`
	IsBuildEnv       bool      `json:"is_build_env"`
	Status           string    `json:"status"`
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
