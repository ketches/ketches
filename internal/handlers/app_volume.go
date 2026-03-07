package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListAppVolumes(c *gin.Context) {
	appID := c.Param("appID")
	vols, err := services.ListAppVolumes(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, vols)
}

func CreateAppVolume(c *gin.Context) {
	appID := c.Param("appID")
	var req models.CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	vol, err := services.CreateAppVolume(c.Request.Context(), appID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, vol)
}

func UpdateAppVolume(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	vol, err := services.UpdateAppVolume(c.Request.Context(), id, &req)
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
	api.Success(c, vol)
}

func DeleteAppVolume(c *gin.Context) {
	id := c.Param("id")
	if err := services.DeleteAppVolume(c.Request.Context(), id); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
