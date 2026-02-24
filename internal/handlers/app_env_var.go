package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/services"
)

func ListAppEnvVars(c *gin.Context) {
	appID := c.Param("appID")
	evs, err := services.ListAppEnvVars(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, evs)
}

func CreateAppEnvVar(c *gin.Context) {
	appID := c.Param("appID")
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	ev, err := services.CreateAppEnvVar(appID, req.Key, req.Value)
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
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	ev, err := services.UpdateAppEnvVar(id, req.Value)
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
