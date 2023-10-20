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

type UserUri struct {
	AccountID string `uri:"account_id"`
}

type UserModel struct {
	AccountID         string `json:"account_id"`
	FullName          string `json:"full_name,omitempty"`
	Email             string `json:"email"`
	Phone             string `json:"phone,omitempty"`
	AccessToken       string `json:"access_token,omitempty"`
	RefreshToken      string `json:"refresh_token,omitempty"`
	MustResetPassword bool   `json:"must_reset_password,omitempty"`
}

// type PaginatedFilter struct {
// 	Page  int `json:"page" form:"page"`
// 	Limit int `json:"limit" form:"limit"`
// }

type UserFilter struct {
	SpaceUri            `json:",inline"`
	QueryAndPagedFilter `form:",inline"`
}

type UserSignUpModel struct {
	AccountID         string `json:"account_id" binding:"required,min=4,max=32"`
	FullName          string `json:"full_name"`
	Password          string `json:"password,omitempty" binding:"required,min=8,max=32"`
	Email             string `json:"email" binding:"required,email"`
	Phone             string `json:"phone"`
	Role              string `json:"role"`
	MustResetPassword bool   `json:"must_reset_password,omitempty"`
}

type UserSignInRequest struct {
	AccountID string `json:"account_id" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type UserRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UserSignOutRequest struct {
	AccountID string `json:"account_id" binding:"required"`
}

type UserUpdateRequest struct {
	UserUri  `json:",inline"`
	FullName string `json:"full_name"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

type UserUpdateResponse struct {
	UserModel `json:",inline"`
}

type UserRenameRequest struct {
	UserUri  `json:",inline"`
	Password string `json:"password" binding:"required"`
	NewName  string `json:"new_name" binding:"required"`
}

type UserRenameResponse struct {
	UserModel `json:",inline"`
}

type UserResetPasswordRequest struct {
	UserUri     `json:",inline"`
	Password    string `json:"password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=32"`
}

type DeleteUserRequest struct {
	UserUri  `json:",inline"`
	Password string `json:"password" binding:"required"`
}
