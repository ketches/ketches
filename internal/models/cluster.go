package models

import "time"

type ClusterResponse struct {
	ID                     string     `json:"id"`
	Slug                   string     `json:"slug"`
	Name                   string     `json:"name"`
	Description            string     `json:"description"`
	Enabled                bool       `json:"enabled"`
	KubeConfig             string     `json:"kube_config"`
	GatewayIP              string     `json:"gateway_ip"`
	ConnectionStatus       string     `json:"connection_status"`
	ConnectionStatusReason string     `json:"connection_status_reason,omitempty"`
	LastCheckedAt          *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
}

type ListClusterResponse struct {
	Items      []ClusterResponse  `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type ClusterPublicResponse struct {
	ID               string `json:"id"`
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	ConnectionStatus string `json:"connection_status"`
}

type CreateClusterRequest struct {
	Slug        string `json:"slug" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	KubeConfig  string `json:"kube_config" binding:"required"`
	GatewayIP   string `json:"gateway_ip"`
}

type UpdateClusterRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	KubeConfig  string `json:"kube_config" binding:"required"`
	GatewayIP   string `json:"gateway_ip"`
}

type PingClusterRequest struct {
	KubeConfig string `json:"kube_config" binding:"required"`
}

type UpdateClusterCredentialsRequest struct {
	KubeConfig string `json:"kube_config" binding:"required"`
	GatewayIP  string `json:"gateway_ip"`
}
