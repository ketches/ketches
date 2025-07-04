/*
Copyright 2025 The Ketches Authors.

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

package models

import "github.com/ketches/ketches/internal/api"

type UserModel struct {
	UserID       string `json:"userID"`
	Username     string `json:"username"`
	Role         string `json:"role,omitempty"`
	Email        string `json:"email"`
	Fullname     string `json:"fullname,omitempty"`
	Gender       int8   `json:"gender"`
	Phone        string `json:"phone,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

type UserRef struct {
	UserID   string `json:"userID" gorm:"column:id"`
	Username string `json:"username"`
	Fullname string `json:"fullname,omitempty"`
}

type ListUsersRequest struct {
	api.QueryAndPagedFilter `form:",inline"`
}

type ListUsersResponse struct {
	Total   int64        `json:"total"`
	Records []*UserModel `json:"records"`
}

type GetUserProfileRequest struct {
	UserID string `uri:"userID" binding:"required"`
}

type UserSignUpRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Fullname string `json:"fullname" binding:"required,min=3,max=64"`
	Password string `json:"password,omitempty" binding:"required,min=8,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

type UserSignInRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserSignOutRequest struct {
	UserID string `json:"userID" binding:"required"`
}

type UserUpdateRequest struct {
	UserID   string `json:"-" uri:"userID"`
	Email    string `json:"email" binding:"required,email"`
	Fullname string `json:"fullname"`
	Gender   int8   `json:"gender"`
	Phone    string `json:"phone"`
}

type UserRenameRequest struct {
	UserID      string `json:"-" uri:"userID"`
	Password    string `json:"password" binding:"required"`
	NewUsername string `json:"newUsername" binding:"required"`
}

type UserChangeRoleRequest struct {
	UserID  string `json:"-" uri:"userID"`
	NewRole string `json:"newRole" binding:"required,oneof=admin user"`
}

type UserResetPasswordRequest struct {
	UserID      string `json:"-" uri:"userID"`
	Password    string `json:"password"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=32"`
}

type DeleteUserRequest struct {
	UserID   string `json:"-" uri:"userID"`
	Password string `json:"password" binding:"required"`
}

type GetUserResourcesResponse struct {
	Projects []*ProjectRef `json:"projects"`
	Envs     []*EnvRef     `json:"envs"`
	Apps     []*AppRef     `json:"apps"`
}

type GetAdminResourcesResponse struct {
	Clusters     []*ClusterRef     `json:"clusters"`
	ClusterNodes []*ClusterNodeRef `json:"clusterNodes"`
}
