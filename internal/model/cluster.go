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

package model

import "time"

type ClusterModel struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	KubeConfig  string    `json:"kubeConfig"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AddClusterRequest struct {
	ClusterModel `json:",inline"`
}

type AddClusterResponse struct {
	ClusterModel `json:",inline"`
}

type UpdateClusterRequest struct {
	ClusterModel `json:",inline"`
}

type UpdateClusterResponse struct {
	ClusterModel `json:",inline"`
}
