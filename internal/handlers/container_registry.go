package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListClusterRegistries(c *gin.Context) {
	clusterID := c.Param("clusterID")

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, registries, err := services.ListClusterRegistries(clusterID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := make([]models.ContainerRegistryResponse, 0, len(registries))
	for _, r := range registries {
		res = append(res, services.ToContainerRegistryResponse(&r))
	}
	api.Success(c, models.ListContainerRegistryResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func CreateClusterRegistry(c *gin.Context) {
	clusterID := c.Param("clusterID")

	var req models.CreateContainerRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	registry, err := services.CreateClusterRegistry(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, services.ToContainerRegistryResponse(registry))
}

func ListProjectContainerRegistries(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, registries, err := services.ListProjectContainerRegistries(projectID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := make([]models.ContainerRegistryResponse, 0, len(registries))
	for _, r := range registries {
		res = append(res, services.ToContainerRegistryResponse(&r))
	}
	api.Success(c, models.ListContainerRegistryResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListProjectContainerRegistriesSimple(c *gin.Context) {
	projectID := c.Param("projectID")
	registries, err := services.ListProjectContainerRegistriesSimple(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := []models.SimpleResponse{}
	for _, r := range registries {
		res = append(res, models.SimpleResponse{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Endpoint, // endpoint as description for registries
			Status:      string(r.Provider),
		})
	}

	api.Success(c, res)
}

func CreateProjectContainerRegistry(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.CreateContainerRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	registry, err := services.CreateProjectContainerRegistry(projectID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, services.ToContainerRegistryResponse(registry))
}

func GetContainerRegistry(c *gin.Context) {
	registryID := c.Param("registryID")

	registry, err := services.GetContainerRegistry(registryID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, services.ToContainerRegistryResponse(registry))
}

func UpdateContainerRegistry(c *gin.Context) {
	registryID := c.Param("registryID")

	var req models.UpdateContainerRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	registry, err := services.UpdateContainerRegistry(registryID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToContainerRegistryResponse(registry))
}

func DeleteContainerRegistry(c *gin.Context) {
	registryID := c.Param("registryID")

	if err := services.DeleteContainerRegistry(registryID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func TestContainerRegistry(c *gin.Context) {
	var req models.TestContainerRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	result := services.TestContainerRegistryConnection(&req)
	api.Success(c, result)
}
