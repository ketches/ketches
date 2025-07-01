package models

import (
	"net/http"
	"time"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
)

type AppModel struct {
	AppID            string           `json:"appID"`
	Slug             string           `json:"slug"`
	DisplayName      string           `json:"displayName,omitempty"`
	Description      string           `json:"description,omitempty"`
	WorkloadType     app.WorkloadType `json:"workloadType,omitempty"` // e.g., "deployment", "statefulset", "daemonset"
	Replicas         int32            `json:"replicas,omitempty"`     // Number of replicas for the app
	ContainerImage   string           `json:"containerImage,omitempty"`
	RegistryUsername string           `json:"registryUsername,omitempty"`
	RegistryPassword string           `json:"registryPassword,omitempty"`
	ContainerCommand string           `json:"containerCommand,omitempty"`
	RequestCPU       int32            `json:"requestCPU,omitempty"`    // in milliCPU (e.g., 500 for 0.5 CPU, 1000 for 1 CPU)
	RequestMemory    int32            `json:"requestMemory,omitempty"` // in MiB
	LimitCPU         int32            `json:"limitCPU,omitempty"`      // in milliCPU (e.g., 1000 for 1 CPU, 2000 for 2 CPUs)
	LimitMemory      int32            `json:"limitMemory,omitempty"`   // in MiB
	Edition          string           `json:"edition,omitempty"`
	EnvID            string           `json:"envID,omitempty"`
	EnvSlug          string           `json:"envSlug,omitempty"`
	ProjectID        string           `json:"projectID,omitempty"`
	ProjectSlug      string           `json:"projectSlug,omitempty"`
	ClusterID        string           `json:"clusterID,omitempty"`
	ClusterSlug      string           `json:"clusterSlug,omitempty"`
	ClusterNamespace string           `json:"clusterNamespace,omitempty"`
	ActualReplicas   int32            `json:"actualReplicas,omitempty"` // Number of currently running replicas
	ActualEdition    string           `json:"actualEdition,omitempty"`  // Edition of the currently running app
	Status           string           `json:"status,omitempty"`         // e.g., "undeployed", "starting", "running", "stopped", "stopping"
	CreatedAt        string           `json:"createdAt,omitempty"`      // ISO 8601 format
}

type ListAppsRequest struct {
	api.QueryAndPagedFilter `form:",inline"`
	EnvID                   string `uri:"envID"`
}

type ListAppsResponse struct {
	Total   int64       `json:"total"`
	Records []*AppModel `json:"records"`
}

type AllAppRefsRequest struct {
	EnvID string `uri:"envID" binding:"required"`
}

type GetAppRefRequest struct {
	AppID string `uri:"appID" binding:"required"`
}

type AppRef struct {
	AppID       string `json:"appID" gorm:"column:id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	EnvID       string `json:"envID"`
	ProjectID   string `json:"projectID"`
}

type GetAppRequest struct {
	AppID string `uri:"appID" binding:"required"`
}

type CreateAppRequest struct {
	EnvID            string `uri:"envID"`
	Slug             string `json:"slug" binding:"required,slug"`
	DisplayName      string `json:"displayName" binding:"required"`
	Description      string `json:"description,omitempty"`
	WorkloadType     string `json:"workloadType" binding:"required,oneof=Deployment StatefulSet"`
	RequestCPU       int32  `json:"requestCPU,omitempty"`                                // in milliCPU (e.g., 500 for 0.5 CPU, 1000 for 1 CPU)
	RequestMemory    int32  `json:"requestMemory,omitempty"`                             // in MiB
	LimitCPU         int32  `json:"limitCPU,omitempty"`                                  // in milliCPU (e.g., 1000 for 1 CPU, 2000 for 2 CPUs)
	LimitMemory      int32  `json:"limitMemory,omitempty"`                               // in MiB
	Replicas         int32  `json:"replicas,omitempty" binding:"required,min=1,max=100"` // Number of replicas for the app
	ContainerImage   string `json:"containerImage" binding:"required"`
	RegistryUsername string `json:"registryUsername,omitempty"`
	RegistryPassword string `json:"registryPassword,omitempty"`
	Deploy           bool   `json:"deploy,omitempty"`
}

type UpdateAppRequest struct {
	AppID       string `json:"-" uri:"appID"`
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description,omitempty"`
}

type UpdateAppImageRequest struct {
	AppID            string `json:"-" uri:"appID"`
	ContainerImage   string `json:"containerImage" binding:"required"`
	RegistryUsername string `json:"registryUsername,omitempty"`
	RegistryPassword string `json:"registryPassword,omitempty"`
}

type SetAppCommandRequest struct {
	AppID            string `json:"-" uri:"appID"`
	ContainerCommand string `json:"containerCommand"`
}

type SetAppResourceRequest struct {
	AppID         string `json:"-" uri:"appID"`
	Replicas      int32  `json:"replicas,omitempty" binding:"required,min=1,max=100"` // Number of replicas for the app
	RequestCPU    int32  `json:"requestCPU,omitempty"`                                // in milliCPU (e.g., 500 for 0.5 CPU, 1000 for 1 CPU)
	RequestMemory int32  `json:"requestMemory,omitempty"`                             // in MiB
	LimitCPU      int32  `json:"limitCPU,omitempty"`                                  // in milliCPU
	LimitMemory   int32  `json:"limitMemory,omitempty"`                               // in MiB
}

type DeleteAppRequest struct {
	AppID string `uri:"appID" binding:"required"`
}

type AppActionRequest struct {
	AppID  string        `json:"-" uri:"appID"`
	Action app.AppAction `json:"action" binding:"required"`
}

type AppInstanceContainerModel struct {
	ContainerName string `json:"containerName"`
	Image         string `json:"image"`
	Status        string `json:"status"`
}

type AppInstanceModel struct {
	InstanceName    string                       `json:"instanceName"`
	Status          string                       `json:"status"`
	CreatedAt       time.Time                    `json:"-"`                         // ISO 8601 format
	RunningDuration string                       `json:"runningDuration,omitempty"` // e.g., "5m", "2h30m"
	InstanceIP      string                       `json:"instanceIP"`
	Containers      []*AppInstanceContainerModel `json:"containers"`
	InitContainers  []*AppInstanceContainerModel `json:"initContainers,omitempty"`
	ContainerCount  int                          `json:"containerCount"`
	NodeName        string                       `json:"nodeName"`
	NodeIP          string                       `json:"nodeIP"`
	RequestCPU      string                       `json:"requestCPU,omitempty"`    // e.g., "500m", "1", "2"
	RequestMemory   string                       `json:"requestMemory,omitempty"` // e.g., "256Mi", "512Mi", "1Gi"
	LimitCPU        string                       `json:"limitCPU,omitempty"`      // e.g., "1", "2", "4"
	LimitMemory     string                       `json:"limitMemory,omitempty"`   // e.g., "512Mi", "1Gi", "2Gi"
	Edition         string                       `json:"revision,omitempty"`
}

type ListAppInstancesRequest struct {
	AppID string `json:"-" uri:"appID"`
}

type ListAppInstancesResponse struct {
	AppID     string              `json:"appID"`
	Slug      string              `json:"slug"`
	Edition   string              `json:"revision,omitempty"`
	Instances []*AppInstanceModel `json:"instances"`
}

type GetAppRunningInfoRequest struct {
	AppID          string              `json:"-" uri:"appID"`
	Request        *http.Request       `json:"-" form:"-"`
	ResponseWriter http.ResponseWriter `json:"-" form:"-"`
}

type GetAppRunningInfoResponse struct {
	AppID          string              `json:"appID"`
	Slug           string              `json:"slug"`
	Replicas       int32               `json:"replicas"`       // Desired number of replicas
	ActualReplicas int32               `json:"actualReplicas"` // Number of currently running replicas
	Edition        string              `json:"edition"`
	ActualEdition  string              `json:"actualEdition"` // Edition of the currently running app
	Status         string              `json:"status"`        // e.g., "running", "stopped", "starting", "stopping"
	Instances      []*AppInstanceModel `json:"instances"`
}

type TerminateAppInstanceRequest struct {
	AppID        string `json:"-" uri:"appID"`
	InstanceName string `json:"instanceName" binding:"required"`
}

type ViewAppContainerLogsRequest struct {
	Request        *http.Request       `json:"-" form:"-"`
	ResponseWriter http.ResponseWriter `json:"-" form:"-"`
	AppID          string              `uri:"appID"`
	InstanceName   string              `uri:"instanceName"`
	ContainerName  string              `uri:"containerName"`   // Optional container name to filter logs
	Follow         bool                `form:"follow"`         // Whether to stream logs
	TailLines      int64               `form:"tailLines"`      // Number of log lines to return from the end
	SinceSeconds   int64               `form:"sinceSeconds"`   // Fetch logs since this many seconds ago
	SinceTime      time.Time           `form:"sinceTime"`      // Fetch logs since this time (RFC3339 format)
	Limit          int64               `form:"limit"`          // Number of log lines to return
	ShowTimestamps bool                `form:"showTimestamps"` // Whether to include timestamps in logs
	Previous       bool                `form:"previous"`       // Whether to fetch logs from the previous instance of the container
}

type ExecAppContainerTerminalRequest struct {
	Request        *http.Request       `json:"-" form:"-"`
	ResponseWriter http.ResponseWriter `json:"-" form:"-"`
	AppID          string              `uri:"appID"`
	InstanceName   string              `uri:"instanceName"`
	ContainerName  string              `uri:"containerName"`
}
