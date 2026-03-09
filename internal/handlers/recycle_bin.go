package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
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
	// For non-admin users, filter to their non-viewer projects
	userID := ""
	claims := api.GetClaims(c)
	if claims != nil && claims.Role != app.UserRoleAdmin {
		userID = claims.UserID
	}

	total, rows, err := services.ListDeletedApps(projectID, userID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	// Convert RecycleBinAppRow to RecycleBinAppResponse
	apps := []models.RecycleBinAppResponse{}
	for _, row := range rows {
		apps = append(apps, models.RecycleBinAppResponse{
			ID:             row.ID,
			Slug:           row.Slug,
			Name:           row.Name,
			Description:    row.Description,
			EnvID:          row.EnvID,
			EnvName:        row.EnvName,
			ProjectID:      row.ProjectID,
			ProjectName:    row.ProjectName,
			ProjectSlug:    row.ProjectSlug,
			AppType:        row.AppType,
			ContainerImage: row.ContainerImage,
			DeletedAt:      row.DeletedAt.Time,
		})
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
	// For non-admin users, filter to their non-viewer projects
	userID := ""
	claims := api.GetClaims(c)
	if claims != nil && claims.Role != app.UserRoleAdmin {
		userID = claims.UserID
	}

	total, envs, err := services.ListDeletedEnvs(projectID, userID, req.Page, req.PageSize, req.Search)
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

	result := []models.RecycleBinAppResponse{}
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

// ListRecycleBinProjects lists soft-deleted projects.
func ListRecycleBinProjects(c *gin.Context) {
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()
	// For non-admin users, filter to their non-viewer projects
	userID := ""
	claims := api.GetClaims(c)
	if claims != nil && claims.Role != "admin" {
		userID = claims.UserID
	}

	total, projects, err := services.ListDeletedProjects(userID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ListRecycleBinProjectResponse{
		Items:      projects,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

// RestoreProjects restores soft-deleted projects.
func RestoreProjects(c *gin.Context) {
	var req models.RestoreResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchRestoreProjects(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// PermanentlyDeleteProjects permanently deletes soft-deleted projects.
func PermanentlyDeleteProjects(c *gin.Context) {
	var req models.PermanentlyDeleteResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchPermanentlyDeleteProjects(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// ListRecycleBinCodeRepositories lists soft-deleted code repositories.
func ListRecycleBinCodeRepositories(c *gin.Context) {
	projectID := c.Query("project_id")
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	userID := ""
	claims := api.GetClaims(c)
	if claims != nil && claims.Role != app.UserRoleAdmin {
		userID = claims.UserID
	}

	total, repos, err := services.ListDeletedCodeRepositories(projectID, userID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ListRecycleBinCodeRepositoryResponse{
		Items:      repos,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

// RestoreCodeRepositories restores soft-deleted code repositories.
func RestoreCodeRepositories(c *gin.Context) {
	var req models.RestoreResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchRestoreCodeRepositories(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

// PermanentlyDeleteCodeRepositories permanently deletes soft-deleted code repositories and all associated data.
func PermanentlyDeleteCodeRepositories(c *gin.Context) {
	var req models.PermanentlyDeleteResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchPermanentlyDeleteCodeRepositories(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}
