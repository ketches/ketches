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

// AuditSpecApplyConfiguration represents an declarative configuration of the AuditSpec type for use
// with apply.
type AuditSpecApplyConfiguration struct {
	SourceKey     *string `json:"sourceKey,omitempty"`
	SourceValue   *string `json:"sourceValue,omitempty"`
	RequestMethod *string `json:"requestMethod,omitempty"`
	RequestPath   *string `json:"requestPath,omitempty"`
	Operator      *string `json:"operator,omitempty"`
}

// AuditSpecApplyConfiguration constructs an declarative configuration of the AuditSpec type for use with
// apply.
func AuditSpec() *AuditSpecApplyConfiguration {
	return &AuditSpecApplyConfiguration{}
}

// WithSourceKey sets the SourceKey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SourceKey field is set to the value of the last call.
func (b *AuditSpecApplyConfiguration) WithSourceKey(value string) *AuditSpecApplyConfiguration {
	b.SourceKey = &value
	return b
}

// WithSourceValue sets the SourceValue field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SourceValue field is set to the value of the last call.
func (b *AuditSpecApplyConfiguration) WithSourceValue(value string) *AuditSpecApplyConfiguration {
	b.SourceValue = &value
	return b
}

// WithRequestMethod sets the RequestMethod field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RequestMethod field is set to the value of the last call.
func (b *AuditSpecApplyConfiguration) WithRequestMethod(value string) *AuditSpecApplyConfiguration {
	b.RequestMethod = &value
	return b
}

// WithRequestPath sets the RequestPath field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RequestPath field is set to the value of the last call.
func (b *AuditSpecApplyConfiguration) WithRequestPath(value string) *AuditSpecApplyConfiguration {
	b.RequestPath = &value
	return b
}

// WithOperator sets the Operator field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Operator field is set to the value of the last call.
func (b *AuditSpecApplyConfiguration) WithOperator(value string) *AuditSpecApplyConfiguration {
	b.Operator = &value
	return b
}