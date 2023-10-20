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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpaceSpec defines the desired state of Space
type SpaceSpec struct {
	//+kubebuilder:validation:Required
	Cluster       string                       `json:"cluster,omitempty"`
	DisplayName   string                       `json:"displayName,omitempty"`
	Description   string                       `json:"description,omitempty"`
	Members       map[string]SpaceMemberRoles  `json:"members,omitempty"`
	ResourceQuota *corev1.ResourceRequirements `json:"resourceQuota,omitempty"`
	LimitRange    *LimitRange                  `json:"limitRange,omitempty"`
}

// SpaceMemberRoles is a list of roles for a space member
type SpaceMemberRoles []SpaceMemberRole

// SpaceMemberRole is a role for a space member
type SpaceMemberRole string

const (
	SpaceMemberRoleOwner      SpaceMemberRole = "space-owner"
	SpaceMemberRoleMaintainer SpaceMemberRole = "space-maintainer"
	SpaceMemberRoleViewer     SpaceMemberRole = "space-viewer"
)

func StringSliceToSpaceMemberRoles(roles []string) SpaceMemberRoles {
	var ret SpaceMemberRoles
	for _, role := range roles {
		ret = append(ret, SpaceMemberRole(role))
	}
	return ret
}

func (r SpaceMemberRoles) StringSlice() []string {
	var ret []string
	for _, role := range r {
		ret = append(ret, string(role))
	}
	return ret
}

// type ResourceQuotaList struct {
// 	CPU    resource.Quantity `json:"cpu,omitempty"`
// 	Memory resource.Quantity `json:"memory,omitempty"`
// 	Pods   resource.Quantity `json:"pods,omitempty"`
// }

// type ResourceQuota struct {
// 	Limits   resource.Resource `json:"limits,omitempty"`
// 	Requests ResourceQuotaList `json:"requests,omitempty"`
// }

type LimitRange struct {
	CPU    resource.Quantity `json:"cpu,omitempty"`
	Memory resource.Quantity `json:"memory,omitempty"`
}

// SpaceStatus defines the observed state of Space
type SpaceStatus struct {
	Phase SpacePhase `json:"phase,omitempty"`
	//+kubebuilder:validation:MaxItems=8
	Conditions []Condition `json:"conditions,omitempty"`
	// Conditions       []SpaceCondition `json:"conditions,omitempty"`
	ApplicationCount int                         `json:"applicationCount"`
	Applications     map[string]ApplicationPhase `json:"applications,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".spec.cluster",description="cluster"
//+kubebuilder:printcolumn:name="Applications",type="integer",JSONPath=".status.applicationCount",description="number of applications"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="status"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="age"
// +genclient
// +genclient:nonNamespaced

// Space is the Schema for the spaces API
type Space struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceSpec   `json:"spec,omitempty"`
	Status SpaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SpaceList contains a list of Space
type SpaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Space `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Space{}, &SpaceList{})
}
