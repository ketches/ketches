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

package global

const (
	OwnedResourceLabel      = "ketches.io/owned=true"
	OwnedResourceLabelKey   = "ketches.io/owned"
	LabelTrueValue          = "true"
	LabelFalseValue         = "false"
	BuiltinResourceLabel    = "ketches.io/builtin=true"
	BuiltinResourceLabelKey = "ketches.io/builtin"

	SystemOperator = "system"

	ContextKeyAccountID  = "account_id"
	ContextKeySignInRole = "sign_in_role"
	ContextKeyEmail      = "email"

	BuiltinNamespace = "ketches-system"

	ExtensionHelmRepositoryName = "ketches-extension"
	ExtensionHelmRepositoryUrl  = "https://ketches.github.io/ketches-extension-charts/"
	VeleroExtensionName         = "velero"
	VeleroExtensionChart        = "ketches-extension/velero"
	KubevirtExtensionName       = "kubevirt"
	KubevirtExtensionChart      = "ketches-extension/kubevirt"
)
