package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

var (
	getPlatformBranding    = services.GetPlatformBranding
	updatePlatformBranding = services.UpdatePlatformBranding
)

func GetPlatformBranding(c *gin.Context) {
	branding, err := getPlatformBranding()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, branding)
}

func UpdatePlatformBranding(c *gin.Context) {
	var request models.UpdatePlatformBrandingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	branding, updateErr := updatePlatformBranding(&request, api.GetClaims(c))
	if updateErr != nil {
		api.Error(c, http.StatusBadRequest, updateErr)
		return
	}
	api.Success(c, branding)
}

func GetPublicSignUpSettings(c *gin.Context) {
	enabled, err := services.GetPublicSignUpEnabled()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.PublicSignUpSettingsResponse{Enabled: enabled})
}

func UpdatePublicSignUpSettings(c *gin.Context) {
	var request models.UpdatePublicSignUpSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.UpdatePublicSignUpEnabled(request.Enabled); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.PublicSignUpSettingsResponse{Enabled: request.Enabled})
}
