package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListClusterIntegrations(c *gin.Context) {
	clusterID := c.Param("clusterID")
	projectID := queryProjectID(c)

	if requireClusterProjectAccess(c, projectID, clusterID) == nil {
		return
	}

	integrations, err := services.ListClusterIntegrations(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := make([]models.ClusterIntegrationResponse, 0, len(integrations))
	for _, i := range integrations {
		res = append(res, models.ClusterIntegrationResponse{
			ID:              i.ID,
			ClusterID:       i.ClusterID,
			IntegrationType: string(i.IntegrationType),
			Name:            i.Name,
			Endpoint:        i.Endpoint,
			Namespace:       i.Namespace,
			ServiceName:     i.ServiceName,
			ServicePort:     i.ServicePort,
			Username:        i.Username,
			SkipTLSVerify:   i.SkipTLSVerify,
			Enabled:         i.Enabled,
			CreatedAt:       i.CreatedAt,
		})
	}

	api.Success(c, res)
}

func CreateClusterIntegration(c *gin.Context) {
	clusterID := c.Param("clusterID")

	var req models.CreateClusterIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	integration, err := services.CreateClusterIntegration(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, models.ClusterIntegrationResponse{
		ID:              integration.ID,
		ClusterID:       integration.ClusterID,
		IntegrationType: string(integration.IntegrationType),
		Name:            integration.Name,
		Endpoint:        integration.Endpoint,
		Username:        integration.Username,
		SkipTLSVerify:   integration.SkipTLSVerify,
		Enabled:         integration.Enabled,
		CreatedAt:       integration.CreatedAt,
	})
}

func GetClusterIntegration(c *gin.Context) {
	integrationID := c.Param("integrationID")

	integration, err := services.GetClusterIntegration(integrationID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, models.ClusterIntegrationResponse{
		ID:              integration.ID,
		ClusterID:       integration.ClusterID,
		IntegrationType: string(integration.IntegrationType),
		Name:            integration.Name,
		Endpoint:        integration.Endpoint,
		Username:        integration.Username,
		SkipTLSVerify:   integration.SkipTLSVerify,
		Enabled:         integration.Enabled,
		CreatedAt:       integration.CreatedAt,
	})
}

func UpdateClusterIntegration(c *gin.Context) {
	integrationID := c.Param("integrationID")

	var req models.UpdateClusterIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	integration, err := services.UpdateClusterIntegration(integrationID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ClusterIntegrationResponse{
		ID:              integration.ID,
		ClusterID:       integration.ClusterID,
		IntegrationType: string(integration.IntegrationType),
		Name:            integration.Name,
		Endpoint:        integration.Endpoint,
		Username:        integration.Username,
		SkipTLSVerify:   integration.SkipTLSVerify,
		Enabled:         integration.Enabled,
		CreatedAt:       integration.CreatedAt,
	})
}

func DeleteClusterIntegration(c *gin.Context) {
	integrationID := c.Param("integrationID")

	if err := services.DeleteClusterIntegration(integrationID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}
