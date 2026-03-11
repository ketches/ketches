package models

import "time"

type OperationLogListRequest struct {
	PaginationRequest
	UserID       string `form:"user_id" json:"user_id"`
	Action       string `form:"action" json:"action"`
	ResourceType string `form:"resource_type" json:"resource_type"`
	Sensitivity  string `form:"sensitivity" json:"sensitivity"`
	Status       string `form:"status" json:"status"`
	Start        string `form:"start" json:"start"`
	End          string `form:"end" json:"end"`
	Export       bool   `form:"export" json:"export"`
}

type OperationLogItem struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UserID         string    `json:"user_id,omitempty"`
	Username       string    `json:"username"`
	Action         string    `json:"action"`
	ResourceType   string    `json:"resource_type"`
	ResourceID     string    `json:"resource_id"`
	ProjectID      string    `json:"project_id,omitempty"`
	EnvID          string    `json:"env_id,omitempty"`
	AppID          string    `json:"app_id,omitempty"`
	RepoID         string    `json:"repo_id,omitempty"`
	Status         string    `json:"status"`
	StatusCode     int       `json:"status_code"`
	Sensitivity    string    `json:"sensitivity"`
	RequestSummary string    `json:"request_summary"`
	ClientIP       string    `json:"client_ip"`
}

type OperationLogListResponse struct {
	Items      []OperationLogItem `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type OperationLogSettingsResponse struct {
	RetentionDays int `json:"retention_days"`
}

type UpdateOperationLogSettingsRequest struct {
	RetentionDays int `json:"retention_days" binding:"required"`
}
