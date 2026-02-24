package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/services"
)

func GetHelmOperatorStatus(c *gin.Context) {
	clusterID := c.Param("clusterID")

	status, err := services.GetHelmOperatorStatus(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, status)
}

func InstallHelmOperator(c *gin.Context) {
	clusterID := c.Param("clusterID")

	if err := services.InstallHelmOperator(clusterID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, gin.H{"message": "helm-operator installed successfully"})
}

func UninstallHelmOperator(c *gin.Context) {
	clusterID := c.Param("clusterID")

	if err := services.UninstallHelmOperator(clusterID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, gin.H{"message": "helm-operator uninstalled successfully"})
}
