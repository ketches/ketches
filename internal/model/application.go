/*
Copyright 2023 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

type ApplicationModel struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	Replicas    int32  `json:"replicas,omitempty"`
	Status      string `json:"status,omitempty"`
}

type ApplicationType string

const (
	ApplicationTypeDeployment  ApplicationType = "Deployment"
	ApplicationTypeStatefulSet ApplicationType = "StatefulSet"
	ApplicationTypeDaemonSet   ApplicationType = "DaemonSet"
	ApplicationTypeJob         ApplicationType = "Job"
	ApplicationTypeCronJob     ApplicationType = "CronJob"
)

type ApplicationFilter struct {
	SpaceUri            `json:",inline"`
	QueryAndPagedFilter `form:",inline"`
}

type CreateApplicationRequest struct {
	SpaceUri    `json:",inline"`
	Type        string
	Name        string     `json:"name" binding:"required"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	Replicas    int32      `json:"replicas"`
	Image       string     `json:"image" binding:"required"`
	Command     string     `json:"command"`
	Args        []string   `json:"args"`
	EnvVars     []KeyValue `json:"env_vars"`
	Ports       []Port     `json:"ports"`
}

type CreateApplicationResponse struct {
	ApplicationModel `json:",inline"`
}

type ExportApplicationsRequest struct {
	SpaceUri      `json:",inline"`
	Applications  []string `json:"applications"`
	WithImageData bool     `json:"with_image_data"`
}

type ExportApplicationsResponse struct {
	Body []byte `json:"body"`
}

type ImportApplicationsRequest struct {
	SpaceUri `json:",inline"`
	Body     []byte `json:"body"`
}

type ImportApplicationsResponse struct {
	Message string `json:"message"`
}

type ApplicationBackup struct {
	ID             string `json:"id"`
	CreatedAt      string `json:"created_at"`
	CreatedBy      string `json:"created_by"`
	TotalItems     int64  `json:"total_items"`
	CompletedItems int64  `json:"completed_items"`
	ExpiredAt      string `json:"expired_at"`
	Status         string `json:"status"`
}

type BackupApplicationRequest struct {
	SpaceUri       `json:",inline"`
	ApplicationUri `json:",inline"`
}

type BackupApplicationResponse struct {
	Body []byte `json:"body"`
}

type ListApplicationBackupsRequest struct {
	SpaceUri       `json:",inline"`
	ApplicationUri `json:",inline"`
}

type ListApplicationBackupsResponse struct {
	Backups []ApplicationBackup `json:"backups"`
}

type CreateApplicationBackupScheduleRequest struct {
	SpaceUri       `json:",inline"`
	ApplicationUri `json:",inline"`
	Cron           string `json:"cron"`
}

type CreateApplicationBackupScheduleResponse struct {
}

type RestoreApplicationsRequest struct {
	SpaceUri `json:",inline"`
}

type RestoreApplicationsResponse struct {
	Message string `json:"message"`
}

type GetPodsAndContainersRequest struct {
	SpaceUri       `json:",inline"`
	ApplicationUri `json:",inline"`
}

type GetPodsAndContainersResponse struct {
	Pods []PodContainers `json:"pods"`
}

type PodContainers struct {
	Name       string      `json:"name"`
	Containers []Container `json:"containers"`
}

type Container struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type GetApplicationContainerLogsRequest struct {
	SpaceUri       `json:",inline"`
	ApplicationUri `json:",inline"`
	Pod            string `json:"pod"`
	Container      string `json:"container"`
	Previous       bool   `json:"previous"`
	Follow         bool   `json:"follow"`
	TailLines      int64  `json:"tail_lines"`
	ShowTimestamp  bool   `json:"show_timestamp"`
}

type GetApplicationContainerLogsResponse struct {
	Body []byte `json:"body"`
}
