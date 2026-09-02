package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListAppConfigFiles(c *gin.Context) {
	appID := c.Param("appID")
	cfs, err := services.ListAppConfigFilesForProjectRole(appID, api.GetProjectRole(c))
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, cfs)
}

func CreateAppConfigFile(c *gin.Context) {
	appID := c.Param("appID")
	var req models.CreateConfigFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cf, err := services.CreateAppConfigFile(c.Request.Context(), appID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, cf)
}

func UpdateAppConfigFile(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateConfigFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cf, err := services.UpdateAppConfigFile(c.Request.Context(), id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.Error(c, http.StatusNotFound, err)
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, cf)
}

func DeleteAppConfigFile(c *gin.Context) {
	id := c.Param("id")
	if err := services.DeleteAppConfigFile(c.Request.Context(), id); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
