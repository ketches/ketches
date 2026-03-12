package models

import "time"

// CreateTestCaseRequest is the request body for creating a test case.
type CreateTestCaseRequest struct {
	Title          string `json:"title" binding:"required"`
	SprintID       string `json:"sprint_id"`
	Precondition   string `json:"precondition"`
	Steps          string `json:"steps" binding:"required"`
	ExpectedResult string `json:"expected_result" binding:"required"`
	RequirementID  string `json:"requirement_id"`
	TaskID         string `json:"task_id"`
}

// UpdateTestCaseRequest is the request body for updating a test case.
type UpdateTestCaseRequest struct {
	Title          string `json:"title" binding:"required"`
	SprintID       string `json:"sprint_id"`
	Precondition   string `json:"precondition"`
	Steps          string `json:"steps" binding:"required"`
	ExpectedResult string `json:"expected_result" binding:"required"`
	RequirementID  string `json:"requirement_id"`
	TaskID         string `json:"task_id"`
}

// TestCaseResponse is the response body for a test case.
type TestCaseResponse struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	SprintID       string    `json:"sprint_id"`
	RequirementID  string    `json:"requirement_id"`
	TaskID         string    `json:"task_id"`
	Title          string    `json:"title"`
	Precondition   string    `json:"precondition"`
	Steps          string    `json:"steps"`
	ExpectedResult string    `json:"expected_result"`
	CreatedBy      string    `json:"created_by"`
	UpdatedBy      string    `json:"updated_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ListTestCaseResponse wraps test case list with pagination.
type ListTestCaseResponse struct {
	Items      []TestCaseResponse `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

// CreateTestRunRequest is the request body for creating a test run.
type CreateTestRunRequest struct {
	Status  string `json:"status" binding:"required"`
	Comment string `json:"comment"`
}

// TestRunResponse is the response body for a test run.
type TestRunResponse struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	TestCaseID string    `json:"test_case_id"`
	Status     string    `json:"status"`
	ExecutedBy string    `json:"executed_by"`
	ExecutedAt time.Time `json:"executed_at"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ListTestRunResponse wraps test run list with pagination.
type ListTestRunResponse struct {
	Items      []TestRunResponse  `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}
