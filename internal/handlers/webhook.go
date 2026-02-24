package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/services"
)

func HandleGitWebhook(c *gin.Context) {
	appID := c.Param("appID")
	secret := c.Query("secret")

	if err := services.HandleGitWebhook(c, appID, secret); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	api.Success(c, gin.H{"message": "webhook received"})
}

func HandleGitWebhookForCodeRepo(c *gin.Context) {
	repoID := c.Param("repoID")
	secret := c.Query("secret")

	if err := services.HandleGitWebhookForCodeRepo(c, repoID, secret); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	api.Success(c, gin.H{"message": "webhook received"})
}
