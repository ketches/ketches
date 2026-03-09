package models

import (
	"time"

	"github.com/ketches/ketches/internal/db/entities"
)

type SimpleApp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type CreateAppRequest struct {
	Slug             string `json:"slug"`
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	AppType          string `json:"app_type"`
	ContainerImage   string `json:"container_image" binding:"required"`
	RegistryUsername string `json:"registry_username"`
	RegistryPassword string `json:"registry_password"`
	Replicas         int    `json:"replicas"`
	Deploy           bool   `json:"deploy"`

	ContainerCommand  string           `json:"container_command"`
	RequestCPU        int              `json:"request_cpu"`
	RequestMemory     int              `json:"request_memory"`
	LimitCPU          int              `json:"limit_cpu"`
	LimitMemory       int              `json:"limit_memory"`
	AutoScaling       *AutoScalingSpec `json:"auto_scaling"`
	SchedulingRule    *SchedulingSpec  `json:"scheduling_rule"`
	Probes            []ProbeSpec      `json:"probes"`
	Gateways          []GatewaySpec    `json:"gateways"`
	SeedImageMetadata bool             `json:"seed_image_metadata,omitempty"` // if true, attempt to seed app configuration from image metadata; default false
}

type AutoScalingSpec struct {
	MinReplicas             int `json:"min_replicas"`
	MaxReplicas             int `json:"max_replicas"`
	TargetCPUUtilization    int `json:"target_cpu_utilization"`
	TargetMemoryUtilization int `json:"target_memory_utilization"`
}

type SchedulingSpec struct {
	RuleType     string `json:"rule_type"`
	NodeName     string `json:"node_name"`
	NodeSelector string `json:"node_selector"`
	NodeAffinity string `json:"node_affinity"`
	Tolerations  string `json:"tolerations"`
}

type ProbeSpec struct {
	Type                string `json:"type"`
	ProbeMode           string `json:"probe_mode"`
	Enabled             bool   `json:"enabled"`
	HttpGetPath         string `json:"http_get_path"`
	HttpGetPort         int    `json:"http_get_port"`
	TcpSocketPort       int    `json:"tcp_socket_port"`
	ExecCommand         string `json:"exec_command"`
	InitialDelaySeconds int    `json:"initial_delay_seconds"`
	PeriodSeconds       int    `json:"period_seconds"`
	TimeoutSeconds      int    `json:"timeout_seconds"`
	SuccessThreshold    int    `json:"success_threshold"`
	FailureThreshold    int    `json:"failure_threshold"`
}

type GatewaySpec struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	GatewayPort int    `json:"gateway_port"`
	Exposed     bool   `json:"exposed"`
	CertID      string `json:"cert_id"`
}

type CreateConfigFileRequest struct {
	Slug      string `json:"slug" binding:"required"`
	MountPath string `json:"mount_path" binding:"required"`
	Content   string `json:"content" binding:"required"`
	FileMode  string `json:"file_mode"`
}

type UpdateConfigFileRequest struct {
	Slug      string `json:"slug" binding:"required"`
	MountPath string `json:"mount_path" binding:"required"`
	Content   string `json:"content" binding:"required"`
	FileMode  string `json:"file_mode"`
}

type CreateVolumeRequest struct {
	Slug         string `json:"slug" binding:"required"`
	VolumeType   string `json:"volume_type" binding:"required"`
	MountPath    string `json:"mount_path" binding:"required"`
	SubPath      string `json:"sub_path"`
	Capacity     int    `json:"capacity" binding:"required"`
	StorageClass string `json:"storage_class"`
	VolumeMode   string `json:"volume_mode"`
	AccessModes  string `json:"access_modes"`
}

type UpdateVolumeRequest struct {
	Slug         string `json:"slug" binding:"required"`
	VolumeType   string `json:"volume_type" binding:"required"`
	MountPath    string `json:"mount_path" binding:"required"`
	SubPath      string `json:"sub_path"`
	Capacity     int    `json:"capacity" binding:"required"`
	StorageClass string `json:"storage_class"`
	VolumeMode   string `json:"volume_mode"`
	AccessModes  string `json:"access_modes"`
}

type CreateGatewayRequest struct {
	Port        int    `json:"port" binding:"required"`
	Protocol    string `json:"protocol" binding:"required"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	GatewayPort int    `json:"gateway_port"`
	Exposed     bool   `json:"exposed"`
	CertID      string `json:"cert_id"`
}

type UpdateGatewayRequest struct {
	Port        int    `json:"port" binding:"required"`
	Protocol    string `json:"protocol" binding:"required"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	GatewayPort int    `json:"gateway_port"`
	Exposed     bool   `json:"exposed"`
	CertID      string `json:"cert_id"`
}

type AppResponse struct {
	ID               string           `json:"id"`
	Slug             string           `json:"slug"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	EnvID            string           `json:"env_id"`
	Env              *EnvResponse     `json:"env,omitempty"`
	AppType          string           `json:"app_type"`
	CodeRepositoryID string           `json:"code_repository_id,omitempty"` // when set, app was deployed from this code repo
	ContainerImage   string           `json:"container_image"`
	ContainerCommand string           `json:"container_command"`
	RegistryUsername string           `json:"registry_username"`
	RegistryPassword string           `json:"registry_password"`
	Replicas         int              `json:"replicas"`
	RequestCPU       int              `json:"request_cpu"`
	RequestMemory    int              `json:"request_memory"`
	LimitCPU         int              `json:"limit_cpu"`
	LimitMemory      int              `json:"limit_memory"`
	Status           string           `json:"status"`
	AutoScaling      *AutoScalingSpec `json:"auto_scaling"`
	SchedulingRule   *SchedulingSpec  `json:"scheduling_rule"`
	CreatedAt        time.Time        `json:"created_at"`
}

type ListAppResponse struct {
	Items      []AppResponse      `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type AppInstanceResponse struct {
	InstanceName       string    `json:"instanceName"`
	Status             string    `json:"status"`
	IP                 string    `json:"ip"`
	InitContainerCount int       `json:"initContainerCount"`
	InitContainers     []string  `json:"initContainers"`
	ContainerCount     int       `json:"containerCount"`
	Containers         []string  `json:"containers"`
	NodeName           string    `json:"nodeName"`
	NodeIP             string    `json:"nodeIP"`
	EventCount         int       `json:"eventCount"`
	RestartCount       int       `json:"restartCount"`
	RunningDuration    string    `json:"runningDuration"`
	CreatedAt          time.Time `json:"createdAt"`
}

type AppEventResponse struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	From      string    `json:"from"`
	Count     int32     `json:"count"`
	CreatedAt time.Time `json:"createdAt"`
}

type AppActionRequest struct {
	Action string `json:"action" binding:"required"`
}

type AppActionResponse struct {
	Status string `json:"status"`
}

type ActionMetadata struct {
	Action   string `json:"action"`
	Label    string `json:"label"`
	Icon     string `json:"icon"`
	Category string `json:"category"`
	Variant  string `json:"variant"`
}

type AvailableActionsResponse struct {
	Actions []ActionMetadata `json:"actions"`
}

type AppTopologyNode struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"` // e.g., "Application", "Deployment", "StatefulSet", "Pod", "Service", "Ingress", "ConfigMap", "PVC"
	Name     string            `json:"name"`
	Status   string            `json:"status,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type AppTopologyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type,omitempty"`
}

type AppTopologyResponse struct {
	Nodes []AppTopologyNode `json:"nodes"`
	Edges []AppTopologyEdge `json:"edges"`
}

type UpdateAppImageRequest struct {
	ContainerImage   string `json:"container_image" binding:"required"`
	RegistryUsername string `json:"registry_username"`
	RegistryPassword string `json:"registry_password"`
}

type UpdateAppReplicasRequest struct {
	Replicas int `json:"replicas" binding:"min=0"`
}

type UpdateAppResourcesRequest struct {
	RequestCPU    int `json:"request_cpu"`
	RequestMemory int `json:"request_memory"`
	LimitCPU      int `json:"limit_cpu"`
	LimitMemory   int `json:"limit_memory"`
}

type UpdateAppAutoScalingRequest struct {
	AutoScaling *AutoScalingSpec `json:"auto_scaling"`
}

type UpdateAppHealthRequest struct {
	Probes []ProbeSpec `json:"probes"`
}

type UpdateAppSchedulingRequest struct {
	SchedulingRule *SchedulingSpec `json:"scheduling_rule"`
}

type UpdateAppCommandRequest struct {
	ContainerCommand string `json:"container_command"`
}

// AppListRow is a flattened DTO for listing apps via JOIN queries.
// It avoids GORM Preload by scanning joined fields directly.
type AppListRow struct {
	// App fields
	ID               string    `gorm:"column:id"`
	Slug             string    `gorm:"column:slug"`
	Name             string    `gorm:"column:name"`
	Description      string    `gorm:"column:description"`
	EnvID            string    `gorm:"column:env_id"`
	AppType          string    `gorm:"column:app_type"`
	CodeRepositoryID *string   `gorm:"column:code_repository_id"`
	ContainerImage   string    `gorm:"column:container_image"`
	ContainerCommand string    `gorm:"column:container_command"`
	RegistryUsername string    `gorm:"column:registry_username"`
	RegistryPassword string    `gorm:"column:registry_password"`
	Replicas         int       `gorm:"column:replicas"`
	RequestCPU       int       `gorm:"column:request_cpu"`
	RequestMemory    int       `gorm:"column:request_memory"`
	LimitCPU         int       `gorm:"column:limit_cpu"`
	LimitMemory      int       `gorm:"column:limit_memory"`
	DeployStatus     string    `gorm:"column:deploy_status"`
	CreatedAt        time.Time `gorm:"column:created_at"`

	// Joined Env fields
	EnvName          string `gorm:"column:env_name"`
	EnvSlug          string `gorm:"column:env_slug"`
	ClusterID        string `gorm:"column:cluster_id"`
	ClusterNamespace string `gorm:"column:cluster_namespace"`
	IsBuildEnv       bool   `gorm:"column:is_build_env"`

	// Joined Cluster fields
	ClusterName string `gorm:"column:cluster_name"`
}

// RecycleBinAppRow represents a flattened app record for the recycle bin list
type RecycleBinAppRow struct {
	entities.App
	ProjectID   string `gorm:"column:project_id"`
	EnvName     string `gorm:"column:env_name"`
	ProjectName string `gorm:"column:project_name"`
	ProjectSlug string `gorm:"column:project_slug"`
}
