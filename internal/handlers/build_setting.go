package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func GetAppBuildSetting(c *gin.Context) {
	appID := c.Param("appID")
	s, err := services.GetAppBuildSetting(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, services.ToBuildSettingResponse(s))
}

func UpsertAppBuildSetting(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpsertAppBuildSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	s, err := services.UpsertAppBuildSetting(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, services.ToBuildSettingResponse(s))
}

func DeleteAppBuildSetting(c *gin.Context) {
	appID := c.Param("appID")
	if err := services.DeleteAppBuildSetting(appID); err != nil {
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
	api.Success(c, services.TestGitConnection(&req))
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
