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

// AuditSpec defines the desired state of Audit
type AuditSpec struct {
	//+kubebuilder:validation:Required
	SourceKey string `json:"sourceKey,omitempty"`
	//+kubebuilder:validation:Required
	SourceValue   string `json:"sourceValue,omitempty"`
	RequestMethod string `json:"requestMethod,omitempty"`
	RequestPath   string `json:"requestPath,omitempty"`
	//+kubebuilder:validation:Required
	Operator string `json:"operator,omitempty"`
}

// AuditStatus defines the observed state of Audit
type AuditStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +genclient

// Audit is the Schema for the audits API
type Audit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuditSpec   `json:"spec,omitempty"`
	Status AuditStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AuditList contains a list of Audit
type AuditList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Audit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Audit{}, &AuditList{})
}
