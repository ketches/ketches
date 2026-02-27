package models

import "time"

// ExtensionCatalogItem is the platform-level catalog entry for an OCI-based helm chart.
type ExtensionCatalogItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	OCIUrl      string    `json:"oci_url"`
	IconURL     string    `json:"icon_url,omitempty"`
	Builtin     bool      `json:"builtin"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateExtensionCatalogItemRequest is the request body for adding a catalog item.
type CreateExtensionCatalogItemRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OCIUrl      string `json:"oci_url" binding:"required"`
	IconURL     string `json:"icon_url"`
}

// ExtensionVersionInfo holds a single chart version tag.
type ExtensionVersionInfo struct {
	Version string `json:"version"`
}

// InstalledExtension represents a helm release installed in a cluster.
type InstalledExtension struct {
	Name             string    `json:"name"`
	CatalogItemID    string    `json:"catalog_item_id,omitempty"`
	OCIUrl           string    `json:"oci_url"`
	ChartVersion     string    `json:"chart_version"`
	ReleaseNamespace string    `json:"release_namespace"`
	Status           string    `json:"status"`
	AppVersion       string    `json:"app_version,omitempty"`
	Values           string    `json:"values,omitempty"`
	Revision         int       `json:"revision"`
	CreatedAt        time.Time `json:"created_at"`
}

// InstallExtensionRequest is the request body for installing an extension into a cluster.
type InstallExtensionRequest struct {
	Name             string `json:"name" binding:"required"`
	CatalogItemID    string `json:"catalog_item_id" binding:"required"`
	ChartVersion     string `json:"chart_version"`
	ReleaseNamespace string `json:"release_namespace"`
	CreateNamespace  bool   `json:"create_namespace"`
	Values           string `json:"values"`
}

// UpdateExtensionRequest is the request body for updating an installed extension.
type UpdateExtensionRequest struct {
	ChartVersion string `json:"chart_version,omitempty"`
	Values       string `json:"values,omitempty"`
}

// UpdateExtensionCatalogItemRequest is the request body for updating a catalog item.
type UpdateExtensionCatalogItemRequest struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OCIUrl      string `json:"oci_url"`
	IconURL     string `json:"icon_url"`
}

// InstalledCluster summarises a cluster that has a catalog extension installed.
type InstalledCluster struct {
	ClusterID   string `json:"cluster_id"`
	ClusterName string `json:"cluster_name"`
	ReleaseName string `json:"release_name"`
	Namespace   string `json:"namespace"`
	Version     string `json:"version"`
	Status      string `json:"status"`
}
