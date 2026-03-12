package models

import "time"

// CreateRequirementRequest is the request body for creating a requirement.
type CreateRequirementRequest struct {
	Title               string `json:"title" binding:"required"`
	Description         string `json:"description"`
	Status              string `json:"status" binding:"required"`
	Priority            string `json:"priority" binding:"required"`
	AssigneeID          string `json:"assignee_id"`
	SprintID            string `json:"sprint_id"`
	ParentRequirementID string `json:"parent_requirement_id"`
}

// UpdateRequirementRequest is the request body for updating a requirement.
type UpdateRequirementRequest struct {
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description"`
	Status         string `json:"status" binding:"required"`
	Priority       string `json:"priority" binding:"required"`
	AssigneeID     string `json:"assignee_id"`
	SprintID       string `json:"sprint_id"`
	PlanningStatus string `json:"planning_status"`
}

// RequirementResponse is the response body for a requirement.
type RequirementResponse struct {
	ID                  string    `json:"id"`
	ProjectID           string    `json:"project_id"`
	SprintID            string    `json:"sprint_id"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	Status              string    `json:"status"`
	Priority            string    `json:"priority"`
	AssigneeID          string    `json:"assignee_id"`
	PlanningStatus      string    `json:"planning_status"`
	BacklogRank         int64     `json:"backlog_rank"`
	ParentRequirementID string    `json:"parent_requirement_id"`
	Depth               int       `json:"depth"`
	CreatedBy           string    `json:"created_by"`
	UpdatedBy           string    `json:"updated_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ListRequirementResponse wraps requirement list with pagination.
type ListRequirementResponse struct {
	Items      []RequirementResponse `json:"items"`
	Pagination PaginationResponse    `json:"pagination"`
}

// BacklogReorderRequest is the request body for reordering backlog items.
type BacklogReorderRequest struct {
	Items []BacklogReorderItem `json:"items" binding:"required"`
}

// BacklogReorderItem represents a single item in a reorder operation.
type BacklogReorderItem struct {
	RequirementID string `json:"requirement_id" binding:"required"`
	Rank          int64  `json:"rank" binding:"required"`
}

// BacklogPlanToSprintRequest is the request body for planning backlog items to a sprint.
type BacklogPlanToSprintRequest struct {
	RequirementIDs []string `json:"requirement_ids" binding:"required"`
	SprintID       string   `json:"sprint_id" binding:"required"`
}

// BacklogReturnRequest is the request body for returning items from a sprint to the backlog.
type BacklogReturnRequest struct {
	RequirementIDs []string `json:"requirement_ids" binding:"required"`
}

type RequirementTransitionRequest struct {
	Status string `json:"status" binding:"required"`
}
