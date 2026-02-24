package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func GetBuildConfig(c *gin.Context) {
	appID := c.Param("appID")

	config, err := services.GetBuildConfig(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, services.ToBuildConfigResponse(config))
}

func UpsertBuildConfig(c *gin.Context) {
	appID := c.Param("appID")

	var req models.UpsertBuildConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	config, err := services.UpsertBuildConfig(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToBuildConfigResponse(config))
}

func DeleteBuildConfig(c *gin.Context) {
	appID := c.Param("appID")

	if err := services.DeleteBuildConfig(appID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func TestGitConnection(c *gin.Context) {
	var req models.TestGitConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	result := services.TestGitConnection(&req)
	api.Success(c, result)
}

func ListAvailableContainerRegistries(c *gin.Context) {
	appID := c.Param("appID")

	registries, err := services.ListAvailableRegistriesForApp(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := make([]models.ContainerRegistryResponse, 0, len(registries))
	for _, r := range registries {
		res = append(res, services.ToContainerRegistryResponse(&r))
	}
	api.Success(c, res)
}
