package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ===== AppConfigFile Handlers =====

// @Summary List App Config Files
// @Description List configuration files for an app
// @Tags AppConfigFile
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=[]models.AppConfigFileModel}
// @Router /api/v1/apps/{appID}/config-files [get]
func ListAppConfigFiles(c *gin.Context) {
	var req models.ListAppConfigFilesRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewAppConfigFileService()
	resp, err := s.ListAppConfigFiles(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, resp)
}

// @Summary Create App Config File
// @Description Create a new configuration file for an app
// @Tags AppConfigFile
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param configFile body models.CreateAppConfigFileRequest true "Config File"
// @Success 201 {object} api.Response{data=models.AppConfigFileModel}
// @Router /api/v1/apps/{appID}/config-files [post]
func CreateAppConfigFile(c *gin.Context) {
	var req models.CreateAppConfigFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")

	s := services.NewAppConfigFileService()
	configFile, err := s.CreateAppConfigFile(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Created(c, configFile)
}

// @Summary Update App Config File
// @Description Update a configuration file for an app
// @Tags AppConfigFile
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param configFileID path string true "Config File ID"
// @Param configFile body models.UpdateAppConfigFileRequest true "Config File"
// @Success 200 {object} api.Response{data=models.AppConfigFileModel}
// @Router /api/v1/apps/{appID}/config-files/{configFileID} [put]
func UpdateAppConfigFile(c *gin.Context) {
	var req models.UpdateAppConfigFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	req.ConfigFileID = c.Param("configFileID")

	s := services.NewAppConfigFileService()
	configFile, err := s.UpdateAppConfigFile(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, configFile)
}

// @Summary Delete App Config Files
// @Description Delete configuration files for an app
// @Tags AppConfigFile
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param req body models.DeleteAppConfigFilesRequest true "Config File IDs"
// @Success 204 {object} api.Response{}
// @Router /api/v1/apps/{appID}/config-files [delete]
func DeleteAppConfigFiles(c *gin.Context) {
	var req models.DeleteAppConfigFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")

	s := services.NewAppConfigFileService()
	if err := s.DeleteAppConfigFiles(c, &req); err != nil {
		api.Error(c, err)
		return
	}
	api.NoContent(c)
}