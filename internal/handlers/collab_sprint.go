package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ListSprints returns paginated sprints for a project.
func ListSprints(c *gin.Context) {
	projectID := c.Param("projectID")

	var params models.CollabFilterParams
	if err := c.ShouldBindQuery(&params); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	total, items, err := services.ListSprints(projectID, &params)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	resp := make([]models.SprintResponse, 0, len(items))
	for _, s := range items {
		resp = append(resp, toSprintResponse(&s))
	}

	api.Success(c, models.ListSprintResponse{
		Items:      resp,
		Pagination: models.BuildPaginationResponse(total, params.Page, params.PageSize),
	})
}

// GetSprint returns a single sprint by ID.
func GetSprint(c *gin.Context) {
	projectID := c.Param("projectID")
	sprintID := c.Param("sprintID")

	sprint, err := services.GetSprint(projectID, sprintID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, toSprintResponse(sprint))
}

// CreateSprint creates a new sprint within a project.
func CreateSprint(c *gin.Context) {
	projectID := c.Param("projectID")
	claims := api.GetClaims(c)

	var req models.CreateSprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	sprint, err := services.CreateSprint(projectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, toSprintResponse(sprint))
}

// UpdateSprint updates an existing sprint.
func UpdateSprint(c *gin.Context) {
	projectID := c.Param("projectID")
	sprintID := c.Param("sprintID")
	claims := api.GetClaims(c)

	var req models.UpdateSprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	sprint, err := services.UpdateSprint(projectID, sprintID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, toSprintResponse(sprint))
}

// DeleteSprint soft-deletes a sprint.
func DeleteSprint(c *gin.Context) {
	projectID := c.Param("projectID")
	sprintID := c.Param("sprintID")

	if err := services.DeleteSprint(projectID, sprintID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// toSprintResponse converts a CollabSprint entity to a SprintResponse.
func toSprintResponse(s *entities.CollabSprint) models.SprintResponse {
	return models.SprintResponse{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Name:      s.Name,
		Goal:      s.Goal,
		Status:    s.Status,
		StartDate: s.StartDate,
		EndDate:   s.EndDate,
		CreatedBy: s.CreatedBy,
		UpdatedBy: s.UpdatedBy,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
