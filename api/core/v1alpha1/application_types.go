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

package v1alpha1

import (
	"github.com/ketches/ketches/api/spec"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	spec.ViewSpec `json:",inline"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;CronJob;Job
	Type WorkloadType `json:"type,omitempty"`
	// +kubebuilder:validation:Enum=Running;Stopped
	DesiredState     ApplicationDesiredState `json:"desiredState,omitempty"`
	Image            string                  `json:"image,omitempty"`
	ImagePullSecrets []string                `json:"imagePullSecret,omitempty"`
	Replicas         int32                   `json:"replicas,omitempty"`
	// Schedule is a cron expression, e.g. "0 0 * * *" for every day at midnight
	// only used for CronJob workload type
	// +optional
	CronSchedule     string                      `json:"cronSchedule,omitempty"`
	Command          []string                    `json:"command,omitempty"`
	Args             []string                    `json:"args,omitempty"`
	Env              []corev1.EnvVar             `json:"env,omitempty"`
	Resources        corev1.ResourceRequirements `json:"resources,omitempty"`
	Healthz          *corev1.Probe               `json:"healthz,omitempty"`
	Autoscaler       *Autoscaler                 `json:"autoscaler,omitempty"`
	Sidecars         []*Sidecar                  `json:"sidecars,omitempty"`
	Ports            []*Port                     `json:"ports,omitempty"`
	MountFiles       []*MountFile                `json:"mountFiles,omitempty"`
	MountDirectories []*MountDirectory           `json:"mountDirectories,omitempty"`
	Privileged       bool                        `json:"privileged,omitempty"`
}

type ApplicationDesiredState string

const (
	ApplicationDesiredStateRunning ApplicationDesiredState = "Running"
	ApplicationDesiredStateStopped ApplicationDesiredState = "Stopped"
)

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	Edition               string                        `json:"edition,omitempty"`
	Phase                 ApplicationPhase              `json:"phase,omitempty"`
	Conditions            []Condition                   `json:"conditions,omitempty"`
	DeploymentConditions  []appsv1.DeploymentCondition  `json:"deploymentConditions,omitempty"`
	StatefulSetConditions []appsv1.StatefulSetCondition `json:"statefulSetConditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=app
// +kubebuilder:printcolumn:name="Workload-Type",type="string",JSONPath=".spec.type",description="workload type"
// +kubebuilder:printcolumn:name="Edition",type="string",JSONPath=".status.edition",description="edition"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="age"
// +genclient

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}

func (app *Application) Workload() client.Object {
	return nil
}

type WorkloadType string

const (
	WorkloadTypeDeployment  WorkloadType = "Deployment"
	WorkloadTypeStatefulSet WorkloadType = "StatefulSet"
	WorkloadTypeCronJob     WorkloadType = "CronJob"
	WorkloadTypeJob         WorkloadType = "Job"
)

type Port struct {
	Number   int32     `json:"number,omitempty"`
	Target   int32     `json:"target,omitempty"`
	Gateways []Gateway `json:"gateways,omitempty"`
}

type Gateway struct {
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Enum=TCP;HTTP
	Type      GatewayType `json:"type,omitempty"`
	ClassName string      `json:"className,omitempty"`
	NodePort  int32       `json:"nodePort,omitempty"`
	Host      string      `json:"host,omitempty"`
	Path      string      `json:"path,omitempty"`
}

type GatewayType string

const (
	// GatewayTypeTCP will expose the container port via an NodePort type service
	GatewayTypeTCP GatewayType = "TCP"
	// GatewayTypeHTTP will expose the container port via a service and an ingress or gateway api
	GatewayTypeHTTP GatewayType = "HTTP"
)

type MountFile struct {
	// IsReferenced bool   `json:"isReferenced,omitempty"`
	Name string `json:"name,omitempty"`
	// Shared bool   `json:"shared,omitempty"`
	// File    string `json:"file,omitempty"`
	Path string `json:"path,omitempty"`
	// Mode is the file mode of the file to be created, default is MountFileModeReadWrite: 0644
	Mode    *int32 `json:"mode,omitempty"`
	Content string `json:"content,omitempty"`
}

type MountDirectory struct {
	Name             string            `json:"name,omitempty"`
	Path             string            `json:"path,omitempty"`
	StorageCapacity  resource.Quantity `json:"storageCapacity,omitempty"`
	StorageClassName *string           `json:"storageClassName,omitempty"`
	Local            bool              `json:"local,omitempty"`
	ReadOnly         bool              `json:"readOnly,omitempty"`
}

type Autoscaler struct {
	MinReplicas                    int32 `json:"minReplicas,omitempty"`
	MaxReplicas                    int32 `json:"maxReplicas,omitempty"`
	TargetCPUUtilizationPercentage int32 `json:"targetCPUUtilizationPercentage,omitempty"`
}

type Sidecar struct {
	// +kubebuilder:validation:Enum=InitRun;PreRun;PostRun
	Type       SidecarType                 `json:"type,omitempty"`
	Name       string                      `json:"name,omitempty"`
	Image      string                      `json:"image,omitempty"`
	Env        []corev1.EnvVar             `json:"env,omitempty"`
	Command    []string                    `json:"command,omitempty"`
	Args       []string                    `json:"args,omitempty"`
	Resources  corev1.ResourceRequirements `json:"resources,omitempty"`
	Privileged bool                        `json:"privileged,omitempty"`
}

type SidecarType string

const (
	// SidecarTypeInitRun will set as init container
	SidecarTypeInitRun SidecarType = "InitRun"
	// SidecarTypePreRun will set before main container
	SidecarTypePreRun SidecarType = "PreRun"
	// SidecarTypePostRun will set after main container
	SidecarTypePostRun SidecarType = "PostRun"
)
