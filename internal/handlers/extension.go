package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// Helm Repository handlers

func ListHelmRepositories(c *gin.Context) {
	clusterID := c.Param("clusterID")
	repos, err := services.ListHelmRepositories(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, repos)
}

func GetHelmRepository(c *gin.Context) {
	clusterID := c.Param("clusterID")
	name := c.Param("repoName")
	repo, err := services.GetHelmRepository(clusterID, name)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, repo)
}

func CreateHelmRepository(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.CreateHelmRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	repo, err := services.CreateHelmRepository(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, repo)
}

func DeleteHelmRepository(c *gin.Context) {
	clusterID := c.Param("clusterID")
	name := c.Param("repoName")
	if err := services.DeleteHelmRepository(clusterID, name); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// GetChartValues returns default values for a chart version in a repo.
func GetChartValues(c *gin.Context) {
	clusterID := c.Param("clusterID")
	repoName := c.Param("repoName")
	chartName := c.Param("chartName")
	version := c.Query("version")
	if version == "" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("version query is required"))
		return
	}
	values, err := services.GetChartValues(clusterID, repoName, chartName, version)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gin.H{"values": values})
}

// Extension (HelmRelease) handlers

func ListExtensions(c *gin.Context) {
	clusterID := c.Param("clusterID")
	extensions, err := services.ListExtensions(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, extensions)
}

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

func UninstallExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	name := c.Param("extensionName")
	if err := services.UninstallExtension(clusterID, name); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
