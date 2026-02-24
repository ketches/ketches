package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListDeploymentHistory(c *gin.Context) {
	appID := c.Param("appID")

	histories, err := services.ListDeploymentHistory(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]*models.DeploymentHistory, len(histories))
	for i, h := range histories {
		result[i] = services.ConvertDeploymentHistoryToModel(&h)
	}

	c.JSON(http.StatusOK, result)
}

func RollbackDeployment(c *gin.Context) {
	appID := c.Param("appID")

	var req models.RollbackDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, err := services.RollbackDeployment(appID, req.HistoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, app)
}
