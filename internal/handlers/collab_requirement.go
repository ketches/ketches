package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ListRequirements returns paginated requirements for a project.
func ListRequirements(c *gin.Context) {
	projectID := c.Param("projectID")

	var params models.CollabFilterParams
	if err := c.ShouldBindQuery(&params); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	total, items, err := services.ListRequirements(projectID, &params)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	resp := make([]models.RequirementResponse, 0, len(items))
	for _, r := range items {
		resp = append(resp, toRequirementResponse(&r))
	}

	api.Success(c, models.ListRequirementResponse{
		Items:      resp,
		Pagination: models.BuildPaginationResponse(total, params.Page, params.PageSize),
	})
}

func ListBacklog(c *gin.Context) {
	projectID := c.Param("projectID")

	var params models.CollabFilterParams
	if err := c.ShouldBindQuery(&params); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	total, items, err := services.ListBacklog(projectID, &params)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	resp := make([]models.RequirementResponse, 0, len(items))
	for _, r := range items {
		resp = append(resp, toRequirementResponse(&r))
	}

	api.Success(c, models.ListRequirementResponse{
		Items:      resp,
		Pagination: models.BuildPaginationResponse(total, params.Page, params.PageSize),
	})
}

// GetRequirement returns a single requirement by ID.
func GetRequirement(c *gin.Context) {
	projectID := c.Param("projectID")
	requirementID := c.Param("requirementID")

	req, err := services.GetRequirement(projectID, requirementID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, toRequirementResponse(req))
}

// CreateRequirement creates a new requirement within a project.
func CreateRequirement(c *gin.Context) {
	projectID := c.Param("projectID")
	claims := api.GetClaims(c)

	var req models.CreateRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	entity, err := services.CreateRequirement(projectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.Created(c, toRequirementResponse(entity))
}

// UpdateRequirement updates an existing requirement.
func UpdateRequirement(c *gin.Context) {
	projectID := c.Param("projectID")
	requirementID := c.Param("requirementID")
	claims := api.GetClaims(c)

	var req models.UpdateRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	entity, err := services.UpdateRequirement(projectID, requirementID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, toRequirementResponse(entity))
}

// DeleteRequirement soft-deletes a requirement.
func DeleteRequirement(c *gin.Context) {
	projectID := c.Param("projectID")
	requirementID := c.Param("requirementID")

	if err := services.DeleteRequirement(projectID, requirementID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func TransitionRequirement(c *gin.Context) {
	projectID := c.Param("projectID")
	requirementID := c.Param("requirementID")
	claims := api.GetClaims(c)

	var req models.RequirementTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	entity, err := services.TransitionRequirement(projectID, requirementID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.Success(c, toRequirementResponse(entity))
}

func CreateRequirementChild(c *gin.Context) {
	projectID := c.Param("projectID")
	parentID := c.Param("requirementID")
	claims := api.GetClaims(c)

	var req models.CreateRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.ParentRequirementID = parentID

	entity, err := services.CreateRequirement(projectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.Created(c, toRequirementResponse(entity))
}

// BacklogReorder reorders backlog items.
func BacklogReorder(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.BacklogReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BacklogReorder(projectID, &req); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// BacklogPlanToSprint moves backlog items into a sprint.
func BacklogPlanToSprint(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.BacklogPlanToSprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BacklogPlanToSprint(projectID, &req); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// BacklogReturn returns items from a sprint back to the backlog.
func BacklogReturn(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.BacklogReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BacklogReturn(projectID, &req); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// toRequirementResponse converts a CollabRequirement entity to a RequirementResponse.
func toRequirementResponse(r *entities.CollabRequirement) models.RequirementResponse {
	return models.RequirementResponse{
		ID:                  r.ID,
		ProjectID:           r.ProjectID,
		SprintID:            r.SprintID,
		Title:               r.Title,
		Description:         r.Description,
		Status:              r.Status,
		Priority:            r.Priority,
		AssigneeID:          r.AssigneeID,
		PlanningStatus:      r.PlanningStatus,
		BacklogRank:         r.BacklogRank,
		ParentRequirementID: r.ParentRequirementID,
		Depth:               r.Depth,
		CreatedBy:           r.CreatedBy,
		UpdatedBy:           r.UpdatedBy,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}
