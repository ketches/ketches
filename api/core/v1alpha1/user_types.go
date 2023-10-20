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

// UserSpec defines the desired state of User
type UserSpec struct {
	Builtin  bool   `json:"builtin,omitempty"`
	FullName string `json:"fullName,omitempty"`
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Pattern="^([a-zA-Z0-9_\\-\\.]+)@([a-zA-Z0-9_\\-\\.]+)\\.([a-zA-Z]{2,5})$"
	Email string `json:"email,omitempty"`
	//+kubebuilder:validation:Pattern="^\\+?[0-9]{10,13}$"
	Phone string `json:"phone,omitempty"`
	//+kubebuilder:validation:Required
	PasswordHash      string `json:"passwordHash,omitempty"`
	MustResetPassword bool   `json:"mustResetPassword,omitempty"`
}

type UserRef struct {
	Name string `json:"name,omitempty"`
}

// UserStatus defines the observed state of User
type UserStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
// +genclient
// +genclient:nonNamespaced

// User is the Schema for the users API
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
