package models

import "github.com/ketches/ketches/internal/api"

type ClusterModel struct {
	ClusterID   string `json:"clusterID"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName,omitempty"`
	KubeConfig  string `json:"kubeConfig,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
}

type ListClustersRequest struct {
	api.QueryAndPagedFilter `form:",inline"`
}

type ListClustersResponse struct {
	Total   int64           `json:"total"`
	Records []*ClusterModel `json:"records"`
}

type GetClusterRefRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type ClusterRef struct {
	ClusterID   string `json:"clusterID" gorm:"column:id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
}

type GetClusterRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type CreateClusterRequest struct {
	Slug        string `json:"slug" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
	KubeConfig  string `json:"kubeConfig" binding:"required"`
	Description string `json:"description,omitempty"`
}

type UpdateClusterRequest struct {
	ClusterID   string `json:"-" uri:"clusterID"`
	DisplayName string `json:"displayName" binding:"required"`
	KubeConfig  string `json:"kubeConfig" binding:"required"`
	Description string `json:"description,omitempty"`
}

type DeleteClusterRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type EnabledClusterRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type DisableClusterRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}
