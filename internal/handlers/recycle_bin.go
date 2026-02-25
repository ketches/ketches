package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListRecycleBinApps(c *gin.Context) {
	projectID := c.Query("project_id")
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, apps, err := services.ListDeletedApps(projectID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ListRecycleBinAppResponse{
		Items:      apps,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListRecycleBinEnvs(c *gin.Context) {
	projectID := c.Query("project_id")
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, envs, err := services.ListDeletedEnvs(projectID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ListRecycleBinEnvResponse{
		Items:      envs,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func RestoreApps(c *gin.Context) {
	var req models.RestoreResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchRestoreApps(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func PermanentlyDeleteApps(c *gin.Context) {
	var req models.PermanentlyDeleteResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchPermanentlyDeleteApps(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func RestoreEnvs(c *gin.Context) {
	var req models.RestoreResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchRestoreEnvs(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func PermanentlyDeleteEnvs(c *gin.Context) {
	var req models.PermanentlyDeleteResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchPermanentlyDeleteEnvs(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func CheckEnvDeletionConflicts(c *gin.Context) {
	envID := c.Param("envID")

	apps, err := services.CheckEnvDeletionConflicts(envID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	var result []models.RecycleBinAppResponse
	for _, app := range apps {
		result = append(result, models.RecycleBinAppResponse{
			ID:             app.ID,
			Slug:           app.Slug,
			Name:           app.Name,
			Description:    app.Description,
			EnvID:          app.EnvID,
			AppType:        app.AppType,
			ContainerImage: app.ContainerImage,
			DeletedAt:      app.DeletedAt.Time,
		})
	}

	api.Success(c, models.EnvDeletionConflictResponse{
		Apps: result,
	})
}
