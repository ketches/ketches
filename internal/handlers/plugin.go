package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func checkProjectAccess(c *gin.Context, projectID string) bool {
	claims, _ := c.Get("claims")
	if claims.(*app.Claims).Role == "admin" {
		return true
	}
	user, _ := c.Get("user")
	if user == nil {
		return false
	}
	hasAccess, err := services.IsProjectMember(projectID, user.(*entities.User).ID)
	if err != nil || !hasAccess {
		api.Error(c, http.StatusForbidden, errors.New("no access to this project"))
		return false
	}
	return true
}

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

	if !checkProjectAccess(c, projectID) {
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
	projectID := c.Param("projectID")

	if !checkProjectAccess(c, projectID) {
		return
	}

	plugin, err := services.GetPlugin(pluginID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, toPluginResponse(plugin))
}

func ListPlugins(c *gin.Context) {
	projectID := c.Param("projectID")

	if !checkProjectAccess(c, projectID) {
		return
	}

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, plugins, err := services.ListProjectPlugins(projectID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	responses := make([]models.PluginResponse, 0, len(plugins))
	for _, plugin := range plugins {
		responses = append(responses, toPluginResponse(&plugin))
	}

	api.Success(c, models.ListPluginResponse{
		Items:      responses,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListPluginsSimple(c *gin.Context) {
	projectID := c.Param("projectID")

	if !checkProjectAccess(c, projectID) {
		return
	}

	plugins, err := services.ListProjectPluginsSimple(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, plugins)
}

func UpdatePlugin(c *gin.Context) {
	pluginID := c.Param("pluginID")
	projectID := c.Param("projectID")

	if !checkProjectAccess(c, projectID) {
		return
	}

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
	projectID := c.Param("projectID")

	if !checkProjectAccess(c, projectID) {
		return
	}

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
	rows, err := services.ListAppPlugins(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	for i := range rows {
		if rows[i].ID == appPlugin.ID {
			api.Created(c, toAppPluginResponse(&rows[i]))
			return
		}
	}
	api.Error(c, http.StatusInternalServerError, errors.New("failed to load installed plugin"))
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
	projectID := c.Param("projectID")

	if !checkProjectAccess(c, projectID) {
		return
	}

	apps, err := services.GetPluginInstalledApps(pluginID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	responses := make([]models.AppResponse, 0, len(apps))
	for _, app := range apps {
		appCtx, err := services.GetAppContext(c.Request.Context(), app.ID)
		if err != nil {
			continue
		}
		responses = append(responses, services.ToAppResponse(c.Request.Context(), appCtx))
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
		ID:                  p.ID,
		Slug:                p.Slug,
		Name:                p.Name,
		Description:         p.Description,
		Image:               p.Image,
		ImagePullPolicy:     p.ImagePullPolicy,
		RegistryUsername:    p.RegistryUsername,
		HasRegistryPassword: strings.TrimSpace(p.RegistryPassword) != "",
		Command:             p.Command,
		EnvVars:             envVars,
		PluginType:          p.PluginType,
		InstallCount:        p.InstallCount,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
}

func toAppPluginResponse(appPlugin *services.AppPluginWithPlugin) models.AppPluginResponse {
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
