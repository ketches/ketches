package models

import "github.com/ketches/ketches/internal/api"

type EnvModel struct {
	EnvID       string `json:"envID"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	ProjectID   string `json:"projectID,omitempty"`
	ClusterID   string `json:"clusterID,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

type ListEnvsRequest struct {
	api.QueryAndPagedFilter `form:",inline"`
	ProjectID               string `uri:"projectID"`
}

type ListEnvsResponse struct {
	Total   int64       `json:"total"`
	Records []*EnvModel `json:"records"`
}

type AllEnvRefsRequest struct {
	ProjectID string `uri:"projectID" binding:"required"`
}

type GetEnvRefRequest struct {
	EnvID string `uri:"envID" binding:"required"`
}

type EnvRef struct {
	EnvID       string `json:"envID" gorm:"column:id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	ProjectID   string `json:"projectID"`
}

type GetEnvRequest struct {
	EnvID string `uri:"envID" binding:"required"`
}

type CreateEnvRequest struct {
	ProjectID   string `uri:"projectID"`
	ClusterID   string `json:"clusterID" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description,omitempty"`
}

type UpdateEnvRequest struct {
	EnvID       string `uri:"envID"`
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description,omitempty"`
}

type DeleteEnvRequest struct {
	EnvID string `uri:"envID" binding:"required"`
}
