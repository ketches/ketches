// Copyright 2025 The Ketches Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

const (
	// LabelPrefix is the global prefix for all Ketches labels.
	LabelPrefix = "ketches.cn/"

	// LabelManagedBy indicates if a resource is managed by Ketches.
	LabelManagedBy = LabelPrefix + "managed"

	// LabelAppID is the unique identifier for an application.
	LabelAppID = LabelPrefix + "app-id"
	// LabelAppSlug is the human-readable slug for an application.
	LabelAppSlug = LabelPrefix + "app-slug"

	// LabelGatewayID is the unique identifier for an application gateway.
	LabelGatewayID = LabelPrefix + "gateway-id"
	// LabelGatewayRouteID is the unique identifier for an application gateway route.
	LabelGatewayRouteID = LabelPrefix + "gateway-route-id"

	// LabelProjectID is the unique identifier for a project.
	LabelProjectID = LabelPrefix + "project-id"
	// LabelProjectSlug is the human-readable slug for a project.
	LabelProjectSlug = LabelPrefix + "project-slug"

	// LabelEnvID is the unique identifier for an environment.
	LabelEnvID = LabelPrefix + "env-id"
	// LabelEnvSlug is the human-readable slug for an environment.
	LabelEnvSlug = LabelPrefix + "env-slug"

	// LabelBuildID is the unique identifier for a build.
	LabelBuildID = LabelPrefix + "build-id"
	// LabelBuildKey indicates a resource is part of a build process.
	LabelBuildKey = LabelPrefix + "build"

	// LabelDebugging indicates an application is in debugging mode.
	LabelDebugging = LabelPrefix + "app-debugging"
)
