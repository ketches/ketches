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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	Envs                   map[string]string `json:"envs,omitempty"`
	GitRepo                string            `json:"gitRepo,omitempty"`
	GitBranch              string            `json:"gitBranch,omitempty"`
	Parameters             map[string]string `json:"parameters,omitempty"`
	Script                 string            `json:"script,omitempty"`
	User                   string            `json:"user,omitempty"`
	Token                  string            `json:"token,omitempty"`
	DockerfilePath         string            `json:"dockerfilePath,omitempty"`
	MaxRetryOnBuildFailure int               `json:"maxRetryOnBuildFailure,omitempty"`
	RollbackOnApplyFailure bool              `json:"rollbackOnFailure,omitempty"`
}

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	Phase      string           `json:"phase,omitempty"`
	Conditions []BuildCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +genclient

// Workflow is the Schema for the workflows API
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

const (
	BuildPhasePending = "Pending"
	BuildPhaseRunning = "Running"
	BuildPhaseSuccess = "Success"
	BuildPhaseFailure = "Failure"
)

type BuildCondition struct {
	Type   string `json:"type,omitempty"`
	Status string `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
}

//+kubebuilder:object:root=true

// WorkflowList contains a list of Workflow
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
