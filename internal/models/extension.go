package models

import "time"

// Helm Repository models

type HelmRepositoryResponse struct {
	Name         string              `json:"name"`
	URL          string              `json:"url"`
	Type         string              `json:"type"`
	Ready        bool                `json:"ready"`
	Message      string              `json:"message,omitempty"`
	Charts       []HelmChartInfo     `json:"charts,omitempty"`
	TotalCharts  int                 `json:"total_charts"`
	LastSyncTime *time.Time          `json:"last_sync_time,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	System       bool                `json:"system"`
}

type HelmChartInfo struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Versions    []HelmChartVersionInfo `json:"versions,omitempty"`
}

type HelmChartVersionInfo struct {
	Version    string     `json:"version"`
	AppVersion string     `json:"app_version,omitempty"`
	Created    *time.Time `json:"created,omitempty"`
}

type CreateHelmRepositoryRequest struct {
	Name     string `json:"name" binding:"required"`
	URL      string `json:"url" binding:"required"`
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Extension (HelmRelease) models

type ExtensionResponse struct {
	Name             string     `json:"name"`
	ChartName        string     `json:"chart_name"`
	ChartVersion     string     `json:"chart_version"`
	Repository       string     `json:"repository,omitempty"`
	OCIRepository    string     `json:"oci_repository,omitempty"`
	ReleaseNamespace string     `json:"release_namespace"`
	ReleaseName      string     `json:"release_name"`
	Status           string     `json:"status"`
	Ready            bool       `json:"ready"`
	Message          string     `json:"message,omitempty"`
	Revision         int        `json:"revision"`
	AppVersion       string     `json:"app_version,omitempty"`
	Suspended        bool       `json:"suspended"`
	Values           string     `json:"values,omitempty"`
	OriginalValues   string     `json:"original_values,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type InstallExtensionRequest struct {
	Name             string `json:"name" binding:"required"`
	ChartName        string `json:"chart_name" binding:"required"`
	ChartVersion     string `json:"chart_version"`
	Repository       string `json:"repository"`
	RepositoryURL    string `json:"repository_url"`
	OCIRepository    string `json:"oci_repository"`
	ReleaseNamespace string `json:"release_namespace"`
	CreateNamespace  bool   `json:"create_namespace"`
	Values           string `json:"values"`
}

type UpdateExtensionRequest struct {
	ChartVersion string `json:"chart_version,omitempty"`
	Values       string `json:"values,omitempty"`
	Suspended    *bool  `json:"suspended,omitempty"`
}
