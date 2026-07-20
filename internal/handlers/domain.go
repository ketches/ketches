package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListClusterDomains(c *gin.Context) {
	clusterID := c.Param("clusterID")
	projectID := queryProjectID(c)
	if requireClusterProjectAccess(c, projectID, clusterID) == nil {
		return
	}

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, items, err := services.ListClusterDomains(clusterID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ListDomainsResponse{
		Items:      toDomainResponses(items),
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListEnvDomains(c *gin.Context) {
	envID := c.Param("envID")

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, items, err := services.ListEnvDomains(envID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ListDomainsResponse{
		Items:      toDomainResponses(items),
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func CreateClusterDomain(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	item, err := services.CreateClusterDomain(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, toDomainResponse(item))
}

func CreateEnvDomain(c *gin.Context) {
	envID := c.Param("envID")
	var req models.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	item, err := services.CreateEnvDomain(envID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, toDomainResponse(item))
}

func UpdateDomain(c *gin.Context) {
	envID := c.Param("envID")
	id := c.Param("domainID")
	var req models.UpdateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	item, err := services.UpdateDomain(envID, id, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, toDomainResponse(item))
}

func DeleteDomain(c *gin.Context) {
	envID := c.Param("envID")
	id := c.Param("domainID")
	if err := services.DeleteDomain(envID, id); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func toDomainResponses(items []entities.Domain) []models.DomainResponse {
	result := make([]models.DomainResponse, 0, len(items))
	for i := range items {
		result = append(result, toDomainResponse(&items[i]))
	}
	return result
}

func toDomainResponse(item *entities.Domain) models.DomainResponse {
	envID := ""
	if item.EnvID != nil {
		envID = *item.EnvID
	}
	return models.DomainResponse{
		ID:          item.ID,
		Name:        item.Name,
		Domain:      item.Domain,
		Description: item.Description,
		Scope:       item.Scope,
		ClusterID:   item.ClusterID,
		EnvID:       envID,
		CreatedAt:   item.CreatedAt,
	}
}
