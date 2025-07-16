package models

import "github.com/ketches/ketches/internal/api"

type ClusterModel struct {
	ClusterID      string `json:"clusterID"`
	Slug           string `json:"slug"`
	DisplayName    string `json:"displayName,omitempty"`
	KubeConfig     string `json:"kubeConfig,omitempty"`
	Description    string `json:"description,omitempty"`
	GatewayIP      string `json:"gatewayIP,omitempty"`
	ReadyNodeCount int    `json:"readyNodeCount,omitempty"`
	NodeCount      int    `json:"nodeCount,omitempty"`
	ServerVersion  string `json:"serverVersion,omitempty"`
	Connectable    bool   `json:"connectable"`
	Enabled        bool   `json:"enabled"`
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
	Slug        string `json:"slug" binding:"required,slug"`
	DisplayName string `json:"displayName" binding:"required"`
	KubeConfig  string `json:"kubeConfig" binding:"required"`
	GatewayIP   string `json:"gatewayIP"`
	Description string `json:"description"`
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

type PingClusterKubeConfigRequest struct {
	KubeConfig string `json:"kubeConfig" binding:"required"`
}

type ListClusterNodesRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type ListClusterNodeRefsRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type GetClusterNodeRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
	NodeName  string `uri:"nodeName" binding:"required"`
}

// ClusterNodeModel corresponds to frontend clusterNodeModel
type ClusterNodeModel struct {
	NodeName                string   `json:"nodeName"`
	Roles                   []string `json:"roles"`
	CreatedAt               string   `json:"createdAt"`
	Version                 string   `json:"version"`
	InternalIP              string   `json:"internalIP"`
	ExternalIP              string   `json:"externalIP"`
	OSImage                 string   `json:"osImage"`
	KernelVersion           string   `json:"kernelVersion"`
	OperatingSystem         string   `json:"operatingSystem"`
	Architecture            string   `json:"architecture"`
	ContainerRuntimeVersion string   `json:"containerRuntimeVersion"`
	KubeletVersion          string   `json:"kubeletVersion"`
	PodCIDR                 string   `json:"podCIDR"`
	Ready                   bool     `json:"ready"`
	ClusterID               string   `json:"clusterID"`
}

type ClusterNodeRef struct {
	NodeName           string `json:"nodeName"`
	NodeIP             string `json:"nodeIP"`
	ClusterID          string `json:"clusterID"`
	ClusterSlug        string `json:"clusterSlug"`
	ClusterDisplayName string `json:"clusterDisplayName"`
}

type ClusterExtensionModel struct {
	ExtensionID   string   `json:"extensionID"`
	Slug          string   `json:"slug"`
	DisplayName   string   `json:"displayName"`
	Description   string   `json:"description,omitempty"`
	Installed     bool     `json:"installed"`
	Version       string   `json:"version,omitempty"`
	Versions      []string `json:"versions,omitempty"`
	InstallMethod string   `json:"installMethod,omitempty"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type ListClusterExtensionsRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type EnableClusterExtensionRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type ListClusterExtensionsResponse = map[string]*ClusterExtensionModel

type ListClusterNodeLabelsRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type ClusterNodeLabelModel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ListClusterNodeTaintsRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type ClusterNodeTaintModel struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type CheckClusterExtensionFeatureEnabledRequest struct {
	ClusterID string `uri:"clusterID" binding:"required"`
}

type InstallClusterExtensionRequest struct {
	ClusterID       string `json:"-" uri:"clusterID"`
	ExtensionName   string `json:"extensionName" binding:"required"`
	Type            string `json:"type" binding:"required,oneof=helm"`
	Version         string `json:"version,omitempty"`
	Values          string `json:"values,omitempty"`
	Namespace       string `json:"namespace,omitempty"`
	CreateNamespace bool   `json:"createNamespace,omitempty"`
}

type UninstallClusterExtensionRequest struct {
	ClusterID     string `json:"-" uri:"clusterID"`
	ExtensionName string `json:"-" uri:"extensionName"`
}

type GetClusterExtensionValuesRequest struct {
	ClusterID     string `json:"-" uri:"clusterID"`
	ExtensionName string `json:"-" uri:"extensionName"`
	Version       string `json:"-" uri:"version"`
}

type GetInstalledExtensionValuesRequest struct {
	ClusterID     string `json:"-" uri:"clusterID"`
	ExtensionName string `json:"-" uri:"extensionName"`
}

type UpdateClusterExtensionRequest struct {
	ClusterID     string `json:"-" uri:"clusterID"`
	ExtensionName string `json:"extensionName" binding:"required"`
	Type          string `json:"type" binding:"required,oneof=helm"`
	Version       string `json:"version,omitempty"`
	Values        string `json:"values,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
}
