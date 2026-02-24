package models

import "time"

type DeploymentHistory struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	CreatedAt time.Time `json:"created_at"`

	ImageBefore    string `json:"image_before"`
	ImageAfter     string `json:"image_after"`
	ReplicasBefore int    `json:"replicas_before"`
	ReplicasAfter  int    `json:"replicas_after"`

	RequestCPUBefore    int `json:"request_cpu_before"`
	RequestCPUAfter     int `json:"request_cpu_after"`
	RequestMemoryBefore int `json:"request_memory_before"`
	RequestMemoryAfter  int `json:"request_memory_after"`
	LimitCPUBefore      int `json:"limit_cpu_before"`
	LimitCPUAfter       int `json:"limit_cpu_after"`
	LimitMemoryBefore   int `json:"limit_memory_before"`
	LimitMemoryAfter    int `json:"limit_memory_after"`

	DeployType string `json:"deploy_type"`
	DeployedBy string `json:"deployed_by"`
	Reason     string `json:"reason"`
	Status     string `json:"status"`

	BuildID *string `json:"build_id,omitempty"`
}

type RollbackDeploymentRequest struct {
	HistoryID string `json:"history_id" binding:"required"`
}
