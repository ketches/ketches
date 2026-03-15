package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

var (
	getPlatformUpdateConfig    = services.GetPlatformUpdateConfig
	updatePlatformUpdateConfig = services.UpdatePlatformUpdateConfig
	getPlatformUpdateStatus    = services.GetPlatformUpdateStatus
	checkPlatformUpdate        = services.CheckPlatformUpdate
	triggerPlatformRollout     = services.TriggerPlatformRollout
)

func GetPlatformUpdateConfig(c *gin.Context) {
	cfg, err := getPlatformUpdateConfig()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, cfg)
}

func UpdatePlatformUpdateConfig(c *gin.Context) {
	var req models.PlatformUpdateConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	cfg, err := updatePlatformUpdateConfig(&req, api.GetClaims(c))
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.Success(c, cfg)
}

func GetPlatformUpdateStatus(c *gin.Context) {
	status, err := getPlatformUpdateStatus()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, status)
}

func CheckPlatformUpdate(c *gin.Context) {
	var req models.CheckPlatformUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	status, err := checkPlatformUpdate(&req, api.GetClaims(c))
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.Success(c, status)
}

func TriggerPlatformRollout(c *gin.Context) {
	var req models.TriggerPlatformRolloutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	result, err := triggerPlatformRollout(&req, api.GetClaims(c))
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusAccepted, api.Response{Data: result})
}
