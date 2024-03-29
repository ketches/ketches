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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	spec.ViewSpec   `json:",inline"`
	KubeConfig      string   `json:"kubeConfig,omitempty"`
	WildCardDomains []string `json:"wildCardDomains,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	Phase          ClusterPhase              `json:"phase,omitempty"`
	Server         string                    `json:"server,omitempty"`
	Version        string                    `json:"version,omitempty"`
	Conditions     []Condition               `json:"conditions,omitempty"`
	SpaceCount     int                       `json:"spaceCount"`
	ExtensionCount int                       `json:"extensionCount"`
	Spaces         map[string]SpacePhase     `json:"spaces,omitempty"`
	Extensions     map[string]ExtensionPhase `json:"extensions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Spaces",type="integer",JSONPath=".status.spaceCount",description="number of spaces"
// +kubebuilder:printcolumn:name="Extensions",type="integer",JSONPath=".status.extensionCount",description="number of extensions"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="status"
// +kubebuilder:printcolumn:name="Server",type="string",JSONPath=".status.server",description="server"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="version"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="age"
// +genclient
// +genclient:nonNamespaced

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
