package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ===== AppEnvVar Handlers =====

// @Summary List App Env Vars
// @Description List environment variables for an app
// @Tags AppEnvVar
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=[]models.AppEnvVarModel}
// @Router /api/v1/apps/{appID}/env-vars [get]
func ListAppEnvVars(c *gin.Context) {
	var req models.ListAppEnvVarsRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewAppEnvVarService()
	resp, err := s.ListAppEnvVars(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, resp)
}

// @Summary Create App Env Var
// @Description Create a new environment variable for an app
// @Tags AppEnvVar
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param envVar body models.AppEnvVarModel true "Env Var"
// @Success 201 {object} api.Response{data=models.AppEnvVarModel}
// @Router /api/v1/apps/{appID}/env-vars [post]
func CreateAppEnvVar(c *gin.Context) {
	var req models.CreateAppEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")

	s := services.NewAppEnvVarService()
	envVar, err := s.CreateAppEnvVar(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Created(c, envVar)
}

// @Summary Update App Env Var
// @Description Update an environment variable for an app
// @Tags AppEnvVar
// @Accept json
// @Produce json
// @Param envVarID path string true "Env Var ID"
// @Param envVar body models.AppEnvVarModel true "Env Var"
// @Success 200 {object} api.Response{data=models.AppEnvVarModel}
// @Router /api/v1/apps/env-vars/{envVarID} [put]
func UpdateAppEnvVar(c *gin.Context) {
	var req models.UpdateAppEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	req.EnvVarID = c.Param("envVarID")

	s := services.NewAppEnvVarService()
	envVar, err := s.UpdateAppEnvVar(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, envVar)
}

// @Summary Delete App Env Vars
// @Description Delete environment variables for an app
// @Tags AppEnvVar
// @Accept json
// @Produce json
// @Param req body models.DeleteAppEnvVarsRequest true "EnvVar IDs"
// @Success 204 {object} api.Response{}
// @Router /api/v1/apps/env-vars [delete]
func DeleteAppEnvVars(c *gin.Context) {
	var req models.DeleteAppEnvVarsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")

	s := services.NewAppEnvVarService()
	if err := s.DeleteAppEnvVars(c, &req); err != nil {
		api.Error(c, err)
		return
	}
	api.NoContent(c)
}
