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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// HelmRepositorySpecApplyConfiguration represents an declarative configuration of the HelmRepositorySpec type for use
// with apply.
type HelmRepositorySpecApplyConfiguration struct {
	Description *string `json:"description,omitempty"`
	Url         *string `json:"url,omitempty"`
}

// HelmRepositorySpecApplyConfiguration constructs an declarative configuration of the HelmRepositorySpec type for use with
// apply.
func HelmRepositorySpec() *HelmRepositorySpecApplyConfiguration {
	return &HelmRepositorySpecApplyConfiguration{}
}

// WithDescription sets the Description field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Description field is set to the value of the last call.
func (b *HelmRepositorySpecApplyConfiguration) WithDescription(value string) *HelmRepositorySpecApplyConfiguration {
	b.Description = &value
	return b
}

// WithUrl sets the Url field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Url field is set to the value of the last call.
func (b *HelmRepositorySpecApplyConfiguration) WithUrl(value string) *HelmRepositorySpecApplyConfiguration {
	b.Url = &value
	return b
}