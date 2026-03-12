package models

import "time"

// CreateTaskRequest is the request body for creating a task.
type CreateTaskRequest struct {
	Title         string   `json:"title" binding:"required"`
	Description   string   `json:"description"`
	Status        string   `json:"status" binding:"required"`
	Priority      string   `json:"priority" binding:"required"`
	AssigneeID    string   `json:"assignee_id"`
	DueDate       string   `json:"due_date"`
	EstimateHours *float64 `json:"estimate_hours"`
	RequirementID string   `json:"requirement_id"`
	SprintID      string   `json:"sprint_id"`
	ParentTaskID  string   `json:"parent_task_id"`
}

// UpdateTaskRequest is the request body for updating a task.
type UpdateTaskRequest struct {
	Title         string   `json:"title" binding:"required"`
	Description   string   `json:"description"`
	Status        string   `json:"status" binding:"required"`
	Priority      string   `json:"priority" binding:"required"`
	AssigneeID    string   `json:"assignee_id"`
	DueDate       string   `json:"due_date"`
	EstimateHours *float64 `json:"estimate_hours"`
	RequirementID string   `json:"requirement_id"`
	SprintID      string   `json:"sprint_id"`
}

// TaskTransitionRequest is the request body for a task status transition.
type TaskTransitionRequest struct {
	Status string `json:"status" binding:"required"`
}

// TaskResponse is the response body for a task.
type TaskResponse struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	SprintID      string     `json:"sprint_id"`
	RequirementID string     `json:"requirement_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Status        string     `json:"status"`
	Priority      string     `json:"priority"`
	AssigneeID    string     `json:"assignee_id"`
	DueDate       *time.Time `json:"due_date"`
	EstimateHours *float64   `json:"estimate_hours"`
	ParentTaskID  string     `json:"parent_task_id"`
	Depth         int        `json:"depth"`
	CreatedBy     string     `json:"created_by"`
	UpdatedBy     string     `json:"updated_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ListTaskResponse wraps task list with pagination.
type ListTaskResponse struct {
	Items      []TaskResponse     `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}
