package models

import "time"

// CreateSprintRequest is the request body for creating a sprint.
type CreateSprintRequest struct {
	Name      string `json:"name" binding:"required"`
	Goal      string `json:"goal"`
	Status    string `json:"status" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

// UpdateSprintRequest is the request body for updating a sprint.
type UpdateSprintRequest struct {
	Name      string `json:"name" binding:"required"`
	Goal      string `json:"goal"`
	Status    string `json:"status" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

// SprintResponse is the response body for a sprint.
type SprintResponse struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	Goal      string     `json:"goal"`
	Status    string     `json:"status"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	CreatedBy string     `json:"created_by"`
	UpdatedBy string     `json:"updated_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ListSprintResponse wraps sprint list with pagination.
type ListSprintResponse struct {
	Items      []SprintResponse   `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}
