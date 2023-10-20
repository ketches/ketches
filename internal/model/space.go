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

type SpaceModel struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
}

type GetSpaceResponse struct {
	SpaceModel `json:",inline"`
	Members    []SpaceMemberDetailModel `json:"members"`
}

type ListSpacesResponse struct {
	Spaces []SpaceModel `json:"spaces"`
}

type CreateSpaceRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name,omitempty"`
	Cluster     string `json:"cluster" binding:"required"`
}

type CreateSpaceResponse struct {
	SpaceModel `json:",inline"`
	Members    []SpaceMemberDetailModel `json:"members"`
}

type UpdateSpaceRequest struct {
	SpaceUri    `json:",inline"`
	DisplayName string `json:"display_name,omitempty"`
}

type UpdateSpaceResponse struct {
	SpaceModel `json:",inline"`
	Members    []SpaceMemberDetailModel `json:"members"`
}

type SpaceMemberModel struct {
	AccountID string   `json:"account_id"`
	Roles     []string `json:"roles"`
}

type SpaceMemberDetailModel struct {
	AccountID string   `json:"account_id"`
	FullName  string   `json:"full_name"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
}

type ListSpaceMembersResponse struct {
	Members []SpaceMemberDetailModel `json:"members"`
}

type AddMembersRequest struct {
	SpaceUri `json:",inline"`
	Members  []SpaceMemberModel `json:"members"`
}

type RemoveSpacesRequest struct {
	SpaceUri `json:",inline"`
	Members  []string `json:"members"`
}
