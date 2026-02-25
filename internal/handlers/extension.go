package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// Extension Catalog handlers

// ListExtensionCatalog returns all platform extension catalog items.
func ListExtensionCatalog(c *gin.Context) {
	items, err := services.ListExtensionCatalog()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, items)
}

// CreateExtensionCatalogItem adds a new catalog item (admin only).
func CreateExtensionCatalogItem(c *gin.Context) {
	var req models.CreateExtensionCatalogItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	claims, _ := c.Get("claims")
	userID := claims.(*app.Claims).UserID

	item, err := services.CreateExtensionCatalogItem(&req, userID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, item)
}

// DeleteExtensionCatalogItem removes a catalog item by ID (admin only, builtin protected).
func DeleteExtensionCatalogItem(c *gin.Context) {
	itemID := c.Param("itemID")
	if err := services.DeleteExtensionCatalogItem(itemID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// Version and values handlers

// ListExtensionVersions lists available OCI tags for a catalog item via crane.
func ListExtensionVersions(c *gin.Context) {
	itemID := c.Param("itemID")
	versions, err := services.ListExtensionVersions(itemID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, versions)
}

// GetExtensionValues returns the default values.yaml for a specific chart version.
func GetExtensionValues(c *gin.Context) {
	itemID := c.Param("itemID")
	version := c.Param("version")
	values, err := services.GetExtensionValues(itemID, version)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gin.H{"values": values})
}

// Installed Extension (helm release) handlers

// ListExtensions lists all helm releases installed in a cluster.
func ListExtensions(c *gin.Context) {
	clusterID := c.Param("clusterID")
	extensions, err := services.ListExtensions(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, extensions)
}

// GetExtension returns a single installed helm release by name.
func GetExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	name := c.Param("extensionName")
	ext, err := services.GetExtension(clusterID, name)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, ext)
}

// InstallExtension installs an OCI helm chart into a cluster.
func InstallExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.InstallExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	ext, err := services.InstallExtension(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, ext)
}

// UpdateExtension upgrades an installed helm release.
func UpdateExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	name := c.Param("extensionName")
	var req models.UpdateExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	ext, err := services.UpdateExtension(clusterID, name, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, ext)
}

// UninstallExtension removes an installed helm release from a cluster.
func UninstallExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	name := c.Param("extensionName")
	if err := services.UninstallExtension(clusterID, name); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
