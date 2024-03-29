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
	"time"

	"github.com/ketches/ketches/pkg/global"
)

type ApplicationVersion string

const (
	ClusterLabelKey            = "ketches.io/cluster"
	SpaceLabelKey              = "ketches.io/space"
	ExtensionLabelKey          = "ketches.io/extension"
	HelmRepositoryLabelKey     = "ketches.io/helmrepository"
	ApplicationLabelKey        = "ketches.io/application"
	ApplicationVersionLabelKey = "ketches.io/application-version"
	ApplicationEditionLabelKey = "ketches.io/application-edition"
	ApplicationVersionStable   = "stable"
	ApplicationVersionCanary   = "canary"
	ApplicationGroupLabelKey   = "ketches.io/applicationgroup"
	AuditSourceKeyLableKey     = "ketches.io/audit-source-key"
	AuditSourceValueLableKey   = "ketches.io/audit-source-value"
)

func BuiltinResourceLabels() map[string]string {
	return map[string]string{
		global.OwnedResourceLabelKey:   global.LabelTrueValue,
		global.BuiltinResourceLabelKey: global.LabelTrueValue,
	}
}

func ClusterRequiredLabelSet(cluster string) map[string]string {
	return map[string]string{
		global.OwnedResourceLabelKey: global.LabelTrueValue,
		ClusterLabelKey:              cluster,
	}
}

// CheckOrSetRequiredLabels checks if the required labels are set on the Cluster object.
// If not, it sets the required labels and returns true.
func (cluster *Cluster) CheckOrSetRequiredLabels() bool {
	var result bool
	if cluster.Labels == nil {
		cluster.Labels = make(map[string]string)
	}
	if cluster.Labels[global.OwnedResourceLabelKey] != global.LabelTrueValue {
		cluster.Labels[global.OwnedResourceLabelKey] = global.LabelTrueValue
		result = true
	}
	if cluster.Labels[ClusterLabelKey] != cluster.Name {
		cluster.Labels[ClusterLabelKey] = cluster.Name
		result = true
	}
	return result
}

func SpaceRequiredLabelSet(space string) map[string]string {
	return map[string]string{
		global.OwnedResourceLabelKey: global.LabelTrueValue,
		SpaceLabelKey:                space,
	}
}

// CheckOrSetRequiredLabels checks if the required labels are set on the Space object.
// If not, it sets the required labels and returns true.
func (space *Space) CheckOrSetRequiredLabels() bool {
	var result bool
	if space.Labels == nil {
		space.Labels = make(map[string]string)
	}
	if space.Labels[global.OwnedResourceLabelKey] != global.LabelTrueValue {
		space.Labels[global.OwnedResourceLabelKey] = global.LabelTrueValue
		result = true
	}
	if space.Labels[SpaceLabelKey] != space.Name {
		space.Labels[SpaceLabelKey] = space.Name
		result = true
	}
	if space.Labels[ClusterLabelKey] != space.Spec.Cluster {
		space.Labels[ClusterLabelKey] = space.Spec.Cluster
		result = true
	}
	return result
}

// CheckOrSetRequiredLabels checks if the required labels are set on the Extension object.
// If not, it sets the required labels and returns true.
func (extension *Extension) CheckOrSetRequiredLabels() bool {
	var result bool
	if extension.Labels == nil {
		extension.Labels = make(map[string]string)
	}
	if extension.Labels[global.OwnedResourceLabelKey] != global.LabelTrueValue {
		extension.Labels[global.OwnedResourceLabelKey] = global.LabelTrueValue
		result = true
	}
	if extension.Labels[ExtensionLabelKey] != extension.Name {
		extension.Labels[ExtensionLabelKey] = extension.Name
		result = true
	}
	return result
}

// CheckOrSetRequiredLabels checks if the required labels are set on the HelmRepository object.
// If not, it sets the required labels and returns true.
func (hr *HelmRepository) CheckOrSetRequiredLabels() bool {
	var result bool
	if hr.Labels == nil {
		hr.Labels = make(map[string]string)
	}
	if hr.Labels[global.OwnedResourceLabelKey] != global.LabelTrueValue {
		hr.Labels[global.OwnedResourceLabelKey] = global.LabelTrueValue
		result = true
	}
	if hr.Labels[HelmRepositoryLabelKey] != hr.Name {
		hr.Labels[HelmRepositoryLabelKey] = hr.Name
		result = true
	}
	if hr.Labels[SpaceLabelKey] != hr.Namespace {
		hr.Labels[SpaceLabelKey] = hr.Namespace
		result = true
	}
	return result
}

func applicationRequiredLabelSet(app string) map[string]string {
	return map[string]string{
		global.OwnedResourceLabelKey: global.LabelTrueValue,
		ApplicationLabelKey:          app,
		ApplicationEditionLabelKey:   NewApplicationEditionLabelValue(),
		ApplicationVersionLabelKey:   "stable",
	}
}

func (app *Application) StableLabelSet() map[string]string {
	return map[string]string{
		global.OwnedResourceLabelKey: global.LabelTrueValue,
		SpaceLabelKey:                app.Namespace,
		ApplicationLabelKey:          app.Name,
		ApplicationVersionLabelKey:   ApplicationVersionStable,
	}
}

func (app *Application) CanaryLabelSet() map[string]string {
	return map[string]string{
		global.OwnedResourceLabelKey: global.LabelTrueValue,
		SpaceLabelKey:                app.Namespace,
		ApplicationLabelKey:          app.Name,
		ApplicationVersionLabelKey:   ApplicationVersionCanary,
	}
}

func NewApplicationEditionLabelValue() string {
	return time.Now().Format("20060102150405")
}

// CheckOrSetRequiredLabels checks if the required labels are set on the Application object.
// If not, it sets the required labels and returns true.
func (app *Application) CheckOrSetRequiredLabels() bool {
	var result bool
	if app.Labels == nil {
		app.Labels = make(map[string]string)
	}
	if app.Labels[global.OwnedResourceLabelKey] != global.LabelTrueValue {
		app.Labels[global.OwnedResourceLabelKey] = global.LabelTrueValue
		result = true
	}
	if app.Labels[ApplicationLabelKey] != app.Name {
		app.Labels[ApplicationLabelKey] = app.Name
		result = true
	}
	if app.Labels[SpaceLabelKey] != app.Namespace {
		app.Labels[SpaceLabelKey] = app.Namespace
		result = true
	}
	if _, ok := app.Labels[ApplicationEditionLabelKey]; !ok {
		app.Labels[ApplicationEditionLabelKey] = NewApplicationEditionLabelValue()
		result = true
	}
	return result
}

func (app *Application) UpdateApplicationEditionLabel() {
	if app.Labels == nil {
		app.Labels = applicationRequiredLabelSet(app.Name)
		return
	}
	app.Labels[ApplicationEditionLabelKey] = NewApplicationEditionLabelValue()
}
