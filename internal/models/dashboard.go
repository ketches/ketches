package models

type DashboardStatsResponse struct {
	ClusterCount     int64 `json:"cluster_count,omitempty"`
	ProjectCount     int64 `json:"project_count,omitempty"`
	EnvironmentCount int64 `json:"environment_count,omitempty"`
	ApplicationCount int64 `json:"application_count,omitempty"`
	UserCount        int64 `json:"user_count,omitempty"`
	MemberCount      int64 `json:"member_count,omitempty"`
}

type EnvironmentResourceUsage struct {
	EnvironmentID   string  `json:"environment_id"`
	EnvironmentName string  `json:"environment_name"`
	Namespace       string  `json:"namespace"`
	CPUUsage        float64 `json:"cpu_usage"`
	CPULimit        float64 `json:"cpu_limit"`
	MemoryUsage     float64 `json:"memory_usage"`
	MemoryLimit     float64 `json:"memory_limit"`
	PodCount        int     `json:"pod_count"`
}

type DashboardResourceUsageResponse struct {
	Environments []EnvironmentResourceUsage `json:"environments"`
}

type PrometheusQueryRequest struct {
	Query string `json:"query" binding:"required"`
	Time  string `json:"time"`
}

type PrometheusQueryRangeRequest struct {
	Query string `json:"query" binding:"required"`
	Start string `json:"start" binding:"required"`
	End   string `json:"end" binding:"required"`
	Step  string `json:"step" binding:"required"`
}
