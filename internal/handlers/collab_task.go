package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ListTasks returns paginated tasks for a project.
func ListTasks(c *gin.Context) {
	projectID := c.Param("projectID")

	var params models.CollabFilterParams
	if err := c.ShouldBindQuery(&params); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	total, items, err := services.ListTasks(projectID, &params)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	resp := make([]models.TaskResponse, 0, len(items))
	for _, t := range items {
		resp = append(resp, toTaskResponse(&t))
	}

	api.Success(c, models.ListTaskResponse{
		Items:      resp,
		Pagination: models.BuildPaginationResponse(total, params.Page, params.PageSize),
	})
}

// GetTask returns a single task by ID.
func GetTask(c *gin.Context) {
	projectID := c.Param("projectID")
	taskID := c.Param("taskID")

	task, err := services.GetTask(projectID, taskID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, toTaskResponse(task))
}

// CreateTask creates a new task within a project.
func CreateTask(c *gin.Context) {
	projectID := c.Param("projectID")
	claims := api.GetClaims(c)

	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	task, err := services.CreateTask(projectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.Created(c, toTaskResponse(task))
}

func CreateTaskChild(c *gin.Context) {
	projectID := c.Param("projectID")
	parentID := c.Param("taskID")
	claims := api.GetClaims(c)

	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.ParentTaskID = parentID

	task, err := services.CreateTask(projectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.Created(c, toTaskResponse(task))
}

// UpdateTask updates an existing task.
func UpdateTask(c *gin.Context) {
	projectID := c.Param("projectID")
	taskID := c.Param("taskID")
	claims := api.GetClaims(c)

	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	task, err := services.UpdateTask(projectID, taskID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, toTaskResponse(task))
}

// DeleteTask soft-deletes a task.
func DeleteTask(c *gin.Context) {
	projectID := c.Param("projectID")
	taskID := c.Param("taskID")

	if err := services.DeleteTask(projectID, taskID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// TransitionTask validates and applies a task status transition.
func TransitionTask(c *gin.Context) {
	projectID := c.Param("projectID")
	taskID := c.Param("taskID")
	claims := api.GetClaims(c)

	var req models.TaskTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	task, err := services.TransitionTask(projectID, taskID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.Success(c, toTaskResponse(task))
}

// toTaskResponse converts a CollabTask entity to a TaskResponse.
func toTaskResponse(t *entities.CollabTask) models.TaskResponse {
	return models.TaskResponse{
		ID:            t.ID,
		ProjectID:     t.ProjectID,
		SprintID:      t.SprintID,
		RequirementID: t.RequirementID,
		Title:         t.Title,
		Description:   t.Description,
		Status:        t.Status,
		Priority:      t.Priority,
		AssigneeID:    t.AssigneeID,
		DueDate:       t.DueDate,
		EstimateHours: t.EstimateHours,
		ParentTaskID:  t.ParentTaskID,
		Depth:         t.Depth,
		CreatedBy:     t.CreatedBy,
		UpdatedBy:     t.UpdatedBy,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}
