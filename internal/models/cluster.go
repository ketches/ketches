package models

import "time"

type SimpleCluster struct {
	ID               string `json:"id"`
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Enabled          bool   `json:"enabled"`
	ConnectionStatus string `json:"connection_status"`
}

type ClusterResponse struct {
	ID                       string     `json:"id"`
	Slug                     string     `json:"slug"`
	Name                     string     `json:"name"`
	Description              string     `json:"description"`
	Enabled                  bool       `json:"enabled"`
	ApiServer                string     `json:"api_server,omitempty"`
	GatewayHost              string     `json:"gateway_host"`
	HasKubeConfig            bool       `json:"has_kube_config"`
	HasPrometheusIntegration bool       `json:"has_prometheus_integration"`
	ConnectionStatus         string     `json:"connection_status"`
	ConnectionStatusReason   string     `json:"connection_status_reason,omitempty"`
	LastCheckedAt            *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
}

type ListClusterResponse struct {
	Items      []ClusterResponse  `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type CreateClusterRequest struct {
	Slug        string `json:"slug" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	KubeConfig  string `json:"kube_config" binding:"required"`
	GatewayHost string `json:"gateway_host"`
}

type UpdateClusterRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	KubeConfig  string `json:"kube_config" binding:"required"`
	GatewayHost string `json:"gateway_host"`
}

type PingClusterRequest struct {
	KubeConfig string `json:"kube_config" binding:"required"`
}

type UpdateClusterCredentialsRequest struct {
	KubeConfig  string `json:"kube_config"`
	GatewayHost string `json:"gateway_host"`
}

type CreateClusterGatewayProviderRequest struct {
	DisplayName      string `json:"display_name"`
	GatewayClassName string `json:"gateway_class_name" binding:"required"`
	ControllerName   string `json:"controller_name" binding:"required"`
	MakeDefault      bool   `json:"make_default"`
}

type ClusterGatewayProvider struct {
	ID                 string `json:"id"`
	ClusterID          string `json:"cluster_id"`
	SourceType         string `json:"source_type"`
	DisplayName        string `json:"display_name"`
	GatewayClassName   string `json:"gateway_class_name"`
	ControllerName     string `json:"controller_name"`
	ExtensionID        string `json:"extension_id,omitempty"`
	ClusterExtensionID string `json:"cluster_extension_id,omitempty"`
	IsDefault          bool   `json:"is_default"`
}

type GatewayClassSummary struct {
	Name           string `json:"name"`
	ControllerName string `json:"controller_name"`
	Accepted       bool   `json:"accepted"`
	IsDefault      bool   `json:"is_default"`
}

type UpdateClusterGatewayClassRequest struct {
	GatewayClassName      string `json:"gateway_class_name" binding:"required"`
	GatewayControllerName string `json:"gateway_controller_name"`
	ManagementMode        string `json:"management_mode"`
}

type ClusterServicePortResponse struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	NodePort   int32  `json:"node_port,omitempty"`
}

type ClusterServiceResponse struct {
	Name  string                       `json:"name"`
	Ports []ClusterServicePortResponse `json:"ports"`
}
