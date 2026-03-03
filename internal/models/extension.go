package models

import "time"

// Extension is the platform-level catalog entry for an OCI-based Helm chart.
type Extension struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	OCIUrl       string    `json:"oci_url"`
	IconURL      string    `json:"icon_url,omitempty"`
	Builtin      bool      `json:"builtin"`
	InstallCount int       `json:"install_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateExtensionRequest is the request body for adding a catalog extension.
type CreateExtensionRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OCIUrl      string `json:"oci_url" binding:"required"`
	IconURL     string `json:"icon_url"`
}

// UpdateExtensionRequest is the request body for updating a catalog extension.
type UpdateExtensionRequest struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OCIUrl      string `json:"oci_url"`
	IconURL     string `json:"icon_url"`
}

	// ExtensionVersionInfo holds a single chart version tag.
	type ExtensionVersionInfo struct {
	Version string `json:"version"`
	}

	// ClusterExtension represents an extension installed (or being installed) on a cluster.
	type ClusterExtension struct {
	ID           string    `json:"id"`
	ClusterID    string    `json:"cluster_id"`
	ExtensionID  string    `json:"extension_id"`
	Namespace    string    `json:"namespace"`
	ReleaseName  string    `json:"release_name"`
	Version      string    `json:"version"`
	Values       string    `json:"values,omitempty"`
	Status       string    `json:"status"`
	Phase        string    `json:"phase"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	}

// InstallExtensionRequest is the request body for installing an extension into a cluster.
type InstallExtensionRequest struct {
	ExtensionID     string `json:"extension_id" binding:"required"`
	ReleaseName     string `json:"release_name" binding:"required"`
	Namespace       string `json:"namespace"`
	Version         string `json:"version"`
	CreateNamespace bool   `json:"create_namespace"`
	Values          string `json:"values"`
}

	// UpgradeExtensionRequest is the request body for upgrading an installed cluster extension.
	type UpgradeExtensionRequest struct {
	Version string `json:"version,omitempty"`
	Values  string `json:"values,omitempty"`
}

// InstalledCluster summarises a cluster that has an extension installed.
type InstalledCluster struct {
	ClusterID   string `json:"cluster_id"`
	ClusterName string `json:"cluster_name"`
	ReleaseName string `json:"release_name"`
	Namespace   string `json:"namespace"`
	Version     string `json:"version"`
	Status      string `json:"status"`
}
