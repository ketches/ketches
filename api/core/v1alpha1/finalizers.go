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
	"slices"
)

const (
	SpaceFinalizer          = "spaces.core.ketches.io/finalizer"
	ExtensionFinalizer      = "extensions.core.ketches.io/finalizer"
	HelmRepositoryFinalizer = "helmrepositories.core.ketches.io/finalizer"
	ApplicationFinalizer    = "applications.core.ketches.io/finalizer"
)

func (space *Space) CheckOrSetFinalizers() bool {
	var result bool
	if !slices.Contains(space.Finalizers, SpaceFinalizer) {
		space.Finalizers = append(space.Finalizers, SpaceFinalizer)
		result = true
	}
	return result
}

func (extension *Extension) CheckOrSetFinalizers() bool {
	var result bool
	if !slices.Contains(extension.Finalizers, ExtensionFinalizer) {
		extension.Finalizers = append(extension.Finalizers, ExtensionFinalizer)
		result = true
	}
	return result
}

func (hr *HelmRepository) CheckOrSetFinalizers() bool {
	var result bool
	if !slices.Contains(hr.Finalizers, HelmRepositoryFinalizer) {
		hr.Finalizers = append(hr.Finalizers, HelmRepositoryFinalizer)
		result = true
	}
	return result
}

func (app *Application) CheckOrSetFinalizers() bool {
	var result bool
	if !slices.Contains(app.Finalizers, ApplicationFinalizer) {
		app.Finalizers = append(app.Finalizers, ApplicationFinalizer)
		result = true
	}
	return result
}
