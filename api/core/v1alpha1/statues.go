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

type (
	ClusterPhase        string
	SpacePhase          string
	HelmRepositoryPhase string
	ExtensionPhase      string
	ApplicationPhase    string
)

type Condition struct {
	Type               string                 `json:"type"`
	Status             metav1.ConditionStatus `json:"status"`
	Message            string                 `json:"message"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`
}

func (p ClusterPhase) String() string { return string(p) }

func (p SpacePhase) String() string { return string(p) }

func (p HelmRepositoryPhase) String() string { return string(p) }

func (p ExtensionPhase) String() string { return string(p) }

func (p ApplicationPhase) String() string { return string(p) }

// Phases for resources
const (
	ClusterPhaseConnecting   ClusterPhase = "Connecting"
	ClusterPhaseConnected    ClusterPhase = "Connected"
	ClusterPhaseDisconnected ClusterPhase = "Disconnected"

	SpacePhaseNotReady SpacePhase = "NotReady"
	SpacePhaseReady    SpacePhase = "Ready"

	ExtensionPhasePending   ExtensionPhase = "Pending"
	ExtensionPhaseInstalled ExtensionPhase = "Installed"
	ExtensionPhaseFailed    ExtensionPhase = "Failed"

	HelmRepositoryPhasePending HelmRepositoryPhase = "Pending"
	HelmRepositoryPhaseAdded   HelmRepositoryPhase = "Added"
	HelmRepositoryPhaseFailed  HelmRepositoryPhase = "Failed"

	ApplicationPhasePending  ApplicationPhase = "Pending"
	ApplicationPhaseRunning  ApplicationPhase = "Running"
	ApplicationPhaseStarting ApplicationPhase = "Starting"
	ApplicationPhaseStopped  ApplicationPhase = "Stopped"
	ApplicationPhaseStopping ApplicationPhase = "Stopping"
	ApplicationPhaseRolling  ApplicationPhase = "Rolling"
	ApplicationPhaseAbnormal ApplicationPhase = "Abnormal"
)

// Condition reasons for resource
const (
	ClusterConditionReasonPingPassed          = "PingPassed"
	ClusterConditionReasonPingFailed          = "PingFailed"
	ClusterConditionReasonGatewayApplySuccess = "GatewayApplySuccess"
	ClusterConditionReasonGatewayApplyFailed  = "GatewayApplyFailed"

	ApplicationConditionReasonSpaceNotFound = "SpaceNotFound"
)

// Condition types for resources
const (
	ClusterConditionTypePingPassed   = "PingPassed"
	ClusterConditionTypeReady        = "Ready"
	ClusterConditionTypeGatewayReady = "GatewayReady"

	SpaceConditionTypeClusterReady       = "ClusterReady"
	SpaceConditionTypeNamespaceReady     = "NamespaceReady"
	SpaceConditionTypeResourceQuotaReady = "ResourceQuotaReady"
	SpaceConditionTypeLimitRangeReady    = "LimitRangeReady"

	ExtensionConditionTypeClusterReady          = "ClusterReady"
	ExtensionConditionTypeHelmRepositoryAdded   = "HelmRepositoryAdded"
	ExtensionConditionTypeHelmRepositoryUpdated = "HelmRepositoryUpdated"
	ExtensionConditionTypeHelmChartFetched      = "HelmChartFetched"
	ExtensionConditionTypeHelmChartInstalled    = "HelmChartInstalled"
	ExtensionConditionTypeHelmChartUpgraded     = "HelmChartUpgraded"
	ExtensionConditionTypeHelmChartUninstalled  = "HelmChartUninstalled"
	ExtensionConditionTypeKubeApplied           = "KubeApplied"
	ExtensionConditionTypeKubeDeleted           = "KubeDeleted"

	HelmRepositoryConditionTypeClusterReady          = "ClusterReady"
	HelmRepositoryConditionTypeSpaceReady            = "SpaceReady"
	HelmRepositoryConditionTypeHelmRepositoryAdded   = "HelmRepositoryAdded"
	HelmRepositoryConditionTypeHelmRepositoryUpdated = "HelmRepositoryUpdated"

	ApplicationConditionTypeSpaceReady    = "SpaceReady"
	ApplicationConditionTypeWorkloadReady = "WorkloadReady"
)

func SetStatusCondition(conditions []Condition, condition Condition) []Condition {
	for i, c := range conditions {
		if c.Type == condition.Type {
			conditions[i] = condition
			return conditions
		}
	}
	return append(conditions, condition)
}

func DeleteStatusCondition(conditions []Condition, conditionType string) []Condition {
	for i, v := range conditions {
		if v.Type == conditionType {
			return append(conditions[:i], conditions[i+1:]...)
		}
	}
	return conditions
}

func conditionStatus(v bool) metav1.ConditionStatus {
	if v {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func (c *Cluster) SetStatusCondition(t string, err error) {
	c.Status.Conditions = SetStatusCondition(c.Status.Conditions, Condition{
		Type:               t,
		Status:             conditionStatus(err == nil),
		LastTransitionTime: metav1.Now(),
		Message:            errorMessage(err),
	})
}

func (c *Cluster) SetStatusSpaces(spaces *SpaceList) {
	spaceNames := make(map[string]SpacePhase)
	if spaces != nil {
		for _, space := range spaces.Items {
			spaceNames[space.Name] = space.Status.Phase
		}
	}

	c.Status.Spaces = spaceNames
	c.Status.SpaceCount = len(spaceNames)
}

func (c *Cluster) SetStatusExtensions(extensions *ExtensionList) {
	extensionNames := make(map[string]ExtensionPhase)
	if extensions != nil {
		for _, extension := range extensions.Items {
			extensionNames[extension.Name] = extension.Status.Phase
		}
	}
	c.Status.Extensions = extensionNames
	c.Status.ExtensionCount = len(extensionNames)
}

func (space *Space) SetStatusCondition(t string, err error) {
	space.Status.Conditions = SetStatusCondition(space.Status.Conditions, Condition{
		Type:               t,
		Status:             conditionStatus(err == nil),
		LastTransitionTime: metav1.Now(),
		Message:            errorMessage(err),
	})
}

func (space *Space) SetStatusApplications(apps *ApplicationList) {
	appNames := make(map[string]ApplicationPhase)
	if apps != nil {
		for _, app := range apps.Items {
			appNames[app.Name] = app.Status.Phase
		}
	}
	space.Status.Applications = appNames
	space.Status.ApplicationCount = len(appNames)
}

func (extension *Extension) SetStatusCondition(t string, err error) {
	extension.Status.Conditions = SetStatusCondition(extension.Status.Conditions, Condition{
		Type:               t,
		Status:             conditionStatus(err == nil),
		LastTransitionTime: metav1.Now(),
		Message:            errorMessage(err),
	})
}

func (hr *HelmRepository) SetStatusCondition(t string, err error) {
	hr.Status.Conditions = SetStatusCondition(hr.Status.Conditions, Condition{
		Type:               t,
		Status:             conditionStatus(err == nil),
		LastTransitionTime: metav1.Now(),
		Message:            errorMessage(err),
	})
}

func (app *Application) SetStatusCondition(t string, err error) {
	app.Status.Conditions = SetStatusCondition(app.Status.Conditions, Condition{
		Type:               ApplicationConditionTypeSpaceReady,
		Status:             conditionStatus(err == nil),
		LastTransitionTime: metav1.Now(),
		Message:            errorMessage(err),
	})
}

func errorMessage(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
