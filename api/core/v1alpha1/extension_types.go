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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ExtensionInstallType string

const (
	ExtensionInstallTypeHelm      ExtensionInstallType = "helm"
	ExtensionInstallTypeKubeApply ExtensionInstallType = "kube-apply"
)

// ExtensionSpec defines the desired state of Extension
type ExtensionSpec struct {
	spec.ViewSpec `json:",inline"`
	// //+kubebuilder:validation:Required
	// Cluster string `json:"cluster,omitempty"`
	// +kubebuilder:validation:Required
	TargetNamespace string `json:"targetNamespace,omitempty"`
	// +kubebuilder:validation:Required
	InstallType           ExtensionInstallType   `json:"installType,omitempty"`
	HelmInstallation      *HelmInstallation      `json:"helmInstallation,omitempty"`
	KubeApplyInstallation *KubeApplyInstallation `json:"applyInstallation,omitempty"`
}

type HelmInstallation struct {
	// +kubebuilder:validation:Required
	Repository string `json:"repository,omitempty"`
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Required
	Chart   string            `json:"chart,omitempty"`
	Version string            `json:"version,omitempty"`
	KeyVals map[string]string `json:"keyVals,omitempty"`
	Values  string            `json:"values,omitempty"`
}

type KubeApplyInstallation struct {
	Name      string   `json:"name,omitempty"`
	RemoteUrl string   `json:"remoteUrl,omitempty"`
	Manifests []string `json:"manifests,omitempty"`
}

// ExtensionStatus defines the observed state of Extension
type ExtensionStatus struct {
	Phase       ExtensionPhase `json:"phase,omitempty"`
	Conditions  []Condition    `json:"conditions,omitempty"`
	HelmRelease *HelmRelease   `json:"helmRelease,omitempty"`
}

type HelmRelease struct {
	Version    string `json:"version,omitempty"`
	Chart      string `json:"chart,omitempty"`
	Revision   int    `json:"revision,omitempty"`
	AppVersion string `json:"appVersion,omitempty"`
	Resources  int    `json:"resources,omitempty"`
	Status     string `json:"status,omitempty"`
}

type KubeApplyResult struct {
	Resources int `json:"resources,omitempty"`
	Applyed   int `json:"applyed,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Install-Type",type="string",JSONPath=".spec.installType",description="install type"
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".spec.cluster",description="cluster"
// +kubebuilder:printcolumn:name="Target-Namespace",type="string",JSONPath=".spec.targetNamespace",description="target namespace"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="age"
// +genclient

// Extension is the Schema for the extensions API
type Extension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExtensionSpec   `json:"spec,omitempty"`
	Status ExtensionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ExtensionList contains a list of Extension
type ExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Extension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Extension{}, &ExtensionList{})
}
