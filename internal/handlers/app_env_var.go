package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListAppEnvVars(c *gin.Context) {
	appID := c.Param("appID")
	evs, err := services.ListAppEnvVarsForProjectRole(appID, api.GetProjectRole(c))
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, evs)
}

func CreateAppEnvVar(c *gin.Context) {
	appID := c.Param("appID")
	var req models.AppEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	isSecret := req.IsSecret != nil && *req.IsSecret
	ev, err := services.CreateAppEnvVar(appID, req.Key, req.Value, isSecret)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, ev)
}

func UpdateAppEnvVar(c *gin.Context) {
	id := c.Param("id")
	var req models.AppEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	var ev *models.AppEnvVarResponse
	var err error
	if req.IsSecret == nil {
		ev, err = services.UpdateAppEnvVar(id, req.Value)
	} else {
		ev, err = services.UpdateAppEnvVar(id, req.Value, *req.IsSecret)
	}
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.Error(c, http.StatusNotFound, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, ev)
}

func DeleteAppEnvVar(c *gin.Context) {
	id := c.Param("id")
	if err := services.DeleteAppEnvVar(id); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
