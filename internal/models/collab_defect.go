package models

import "time"

// CreateDefectRequest is the request body for creating a defect.
type CreateDefectRequest struct {
	Title             string `json:"title" binding:"required"`
	Description       string `json:"description" binding:"required"`
	Severity          string `json:"severity" binding:"required"`
	Status            string `json:"status" binding:"required"`
	AssigneeID        string `json:"assignee_id"`
	SprintID          string `json:"sprint_id"`
	RequirementID     string `json:"requirement_id"`
	TaskID            string `json:"task_id"`
	TestCaseID        string `json:"test_case_id"`
	TestRunID         string `json:"test_run_id"`
	ReproductionSteps string `json:"reproduction_steps"`
}

// UpdateDefectRequest is the request body for updating a defect.
type UpdateDefectRequest struct {
	Title              string `json:"title" binding:"required"`
	Description        string `json:"description" binding:"required"`
	Severity           string `json:"severity" binding:"required"`
	Status             string `json:"status" binding:"required"`
	AssigneeID         string `json:"assignee_id"`
	SprintID           string `json:"sprint_id"`
	RequirementID      string `json:"requirement_id"`
	TaskID             string `json:"task_id"`
	TestCaseID         string `json:"test_case_id"`
	TestRunID          string `json:"test_run_id"`
	ReproductionSteps  string `json:"reproduction_steps"`
	FixNote            string `json:"fix_note"`
	RuntimeContextJSON string `json:"runtime_context_json"`
}

// DefectTransitionRequest is the request body for a defect status transition.
type DefectTransitionRequest struct {
	Status string `json:"status" binding:"required"`
}

// DefectResponse is the response body for a defect.
type DefectResponse struct {
	ID                 string    `json:"id"`
	ProjectID          string    `json:"project_id"`
	SprintID           string    `json:"sprint_id"`
	RequirementID      string    `json:"requirement_id"`
	TaskID             string    `json:"task_id"`
	TestCaseID         string    `json:"test_case_id"`
	TestRunID          string    `json:"test_run_id"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Severity           string    `json:"severity"`
	Status             string    `json:"status"`
	AssigneeID         string    `json:"assignee_id"`
	ReproductionSteps  string    `json:"reproduction_steps"`
	FixNote            string    `json:"fix_note"`
	RuntimeContextJSON string    `json:"runtime_context_json"`
	CreatedBy          string    `json:"created_by"`
	UpdatedBy          string    `json:"updated_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ListDefectResponse wraps defect list with pagination.
type ListDefectResponse struct {
	Items      []DefectResponse   `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}
