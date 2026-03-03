package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// === Extension Catalog handlers ===

// ListExtensions returns all platform extension catalog items.
func ListExtensions(c *gin.Context) {
	items, err := services.ListExtensions()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, items)
}

// CreateExtension adds a new catalog extension (admin only).
func CreateExtension(c *gin.Context) {
	var req models.CreateExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	claims, _ := c.Get("claims")
	userID := claims.(*app.Claims).UserID
	item, err := services.CreateExtension(&req, userID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, item)
}

// DeleteExtension removes a catalog extension by ID (admin only, builtin protected).
func DeleteExtension(c *gin.Context) {
	extensionID := c.Param("extensionID")
	if err := services.DeleteExtension(extensionID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// UpdateExtension updates a non-builtin catalog extension's metadata (admin only).
func UpdateExtension(c *gin.Context) {
	extensionID := c.Param("extensionID")
	var req models.UpdateExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	item, err := services.UpdateExtension(extensionID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, item)
}

// GetInstalledClustersForExtension returns all clusters that have a given extension installed.
func GetInstalledClustersForExtension(c *gin.Context) {
	extensionID := c.Param("extensionID")
	clusters, err := services.GetInstalledClustersForExtension(extensionID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, clusters)
}

// ListExtensionVersions lists available OCI tags for an extension.
func ListExtensionVersions(c *gin.Context) {
	extensionID := c.Param("extensionID")
	versions, err := services.ListExtensionVersions(extensionID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, versions)
}

// GetExtensionValues returns the default values.yaml for a specific chart version.
func GetExtensionValues(c *gin.Context) {
	extensionID := c.Param("extensionID")
	version := c.Param("version")
	values, err := services.GetExtensionValues(extensionID, version)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gin.H{"values": values})
}

// === Cluster Extension handlers ===

// ListClusterExtensions lists all extensions installed in a cluster (from DB).
func ListClusterExtensions(c *gin.Context) {
	clusterID := c.Param("clusterID")
	extensions, err := services.ListClusterExtensions(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, extensions)
}

// GetClusterExtension returns a single installed cluster extension by UUID.
func GetClusterExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	id := c.Param("clusterExtensionID")
	ext, err := services.GetClusterExtension(clusterID, id)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, ext)
}

// InstallClusterExtension installs an OCI helm chart into a cluster (async, returns 202).
func InstallClusterExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.InstallExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	claims, _ := c.Get("claims")
	userID := claims.(*app.Claims).UserID
	ext, err := services.InstallClusterExtension(clusterID, &req, userID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusAccepted, ext)
}

// UpgradeClusterExtension upgrades an installed cluster extension (async).
func UpgradeClusterExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	id := c.Param("clusterExtensionID")
	var req models.UpgradeExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	ext, err := services.UpgradeClusterExtension(clusterID, id, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, ext)
}

// UninstallClusterExtension removes an installed helm release from a cluster (async).
func UninstallClusterExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	id := c.Param("clusterExtensionID")
	if err := services.UninstallClusterExtension(clusterID, id); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	// Return the updated record (now status=uninstalling) as 202.
	ext, err := services.GetClusterExtension(clusterID, id)
	if err != nil {
		// Record already hard-deleted (edge case) — return 204.
		api.NoContent(c)
		return
	}
	c.JSON(http.StatusAccepted, ext)
}
