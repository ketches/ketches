package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func CreatePlugin(c *gin.Context) {
	var req models.CreatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	projectID := c.Param("projectID")
	if projectID != "" {
		req.ProjectID = projectID
	}

	if req.ProjectID == "" {
		api.Error(c, http.StatusBadRequest, errors.New("project_id is required"))
		return
	}

	plugin, err := services.CreatePlugin(&req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, toPluginResponse(plugin))
}

func GetPlugin(c *gin.Context) {
	pluginID := c.Param("pluginID")

	plugin, err := services.GetPlugin(pluginID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, toPluginResponse(plugin))
}

func ListPlugins(c *gin.Context) {
	projectID := c.Param("projectID")
	search := c.Query("search")

	var plugins []entities.Plugin
	var err error
	if projectID != "" {
		plugins, err = services.ListProjectPlugins(projectID, search)
	} else {
		plugins, err = services.ListPlugins(search)
	}

	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	responses := make([]models.PluginResponse, 0, len(plugins))
	for _, plugin := range plugins {
		responses = append(responses, toPluginResponse(&plugin))
	}

	api.Success(c, responses)
}

func UpdatePlugin(c *gin.Context) {
	pluginID := c.Param("pluginID")

	var req models.UpdatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	plugin, err := services.UpdatePlugin(pluginID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toPluginResponse(plugin))
}

func DeletePlugin(c *gin.Context) {
	pluginID := c.Param("pluginID")

	if err := services.DeletePlugin(pluginID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func InstallPluginToApp(c *gin.Context) {
	appID := c.Param("appID")

	var req models.InstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	appPlugin, err := services.InstallPluginToApp(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, toAppPluginResponse(appPlugin))
}

func UninstallPluginFromApp(c *gin.Context) {
	appID := c.Param("appID")
	pluginID := c.Param("pluginID")

	if err := services.UninstallPluginFromApp(appID, pluginID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func ListAppPlugins(c *gin.Context) {
	appID := c.Param("appID")

	appPlugins, err := services.ListAppPlugins(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	responses := make([]models.AppPluginResponse, 0, len(appPlugins))
	for _, appPlugin := range appPlugins {
		responses = append(responses, toAppPluginResponse(&appPlugin))
	}

	api.Success(c, responses)
}

func ToggleAppPlugin(c *gin.Context) {
	appID := c.Param("appID")
	pluginID := c.Param("pluginID")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.ToggleAppPlugin(appID, pluginID, req.Enabled); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func UpdateAppPluginEnv(c *gin.Context) {
	appID := c.Param("appID")
	pluginID := c.Param("pluginID")

	var req models.UpdateAppPluginEnvRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.UpdateAppPluginEnvVars(appID, pluginID, req.EnvVars); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, nil)
}

func GetPluginInstalledApps(c *gin.Context) {
	pluginID := c.Param("pluginID")

	apps, err := services.GetPluginInstalledApps(pluginID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	responses := make([]models.AppResponse, 0, len(apps))
	for _, app := range apps {
		responses = append(responses, toAppResponse(c, &app))
	}

	api.Success(c, responses)
}

func toPluginResponse(plugin any) models.PluginResponse {
	p := plugin.(*entities.Plugin)

	var envVars []models.PluginEnvVar
	if p.EnvVars != "" {
		json.Unmarshal([]byte(p.EnvVars), &envVars)
	}

	return models.PluginResponse{
		ID:               p.ID,
		Slug:             p.Slug,
		Name:             p.Name,
		Description:      p.Description,
		Image:            p.Image,
		RegistryUsername: p.RegistryUsername,
		Command:          p.Command,
		EnvVars:          envVars,
		PluginType:       p.PluginType,
		InstallCount:     p.InstallCount,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

func toAppPluginResponse(appPlugin *entities.AppPlugin) models.AppPluginResponse {
	var envVars []models.PluginEnvVar
	if appPlugin.EnvVars != "" {
		json.Unmarshal([]byte(appPlugin.EnvVars), &envVars)
	}

	return models.AppPluginResponse{
		ID:        appPlugin.ID,
		AppID:     appPlugin.AppID,
		PluginID:  appPlugin.PluginID,
		Enabled:   appPlugin.Enabled,
		EnvVars:   envVars,
		Plugin:    toPluginResponse(&appPlugin.Plugin),
		CreatedAt: appPlugin.CreatedAt,
	}
}
