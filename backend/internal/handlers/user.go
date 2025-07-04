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

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// @Summary List Users
// @Description List users
// @Tags User
// @Accept json
// @Produce json
// @Param query query models.ListUsersRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=models.ListUsersResponse}
// @Router /api/v1/users [get]
func ListUsers(c *gin.Context) {
	var req models.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewUserService()
	resp, err := s.List(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary Get User
// @Description Get user by user ID
// @Tags User
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/{userID} [get]
func GetUser(c *gin.Context) {
	var req models.GetUserProfileRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	s := services.NewUserService()
	user, err := s.Get(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, user)
}

// @Summary Sign Up User
// @Description Sign up a new user
// @Tags User
// @Accept json
// @Produce json
// @Param user body models.UserSignUpRequest true "User sign up request"
// @Success 201 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/sign-up [post]
func UserSignUp(c *gin.Context) {
	var req models.UserSignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewUserService()
	user, err := s.SignUp(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Created(c, user)
}

// @Summary Sign In User
// @Description Sign in an existing user
// @Tags User
// @Accept json
// @Produce json
// @Param user body models.UserSignInRequest true "User sign in request"
// @Success 201 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/sign-in [post]
func UserSignIn(c *gin.Context) {
	var req models.UserSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewUserService()
	user, err := s.SignIn(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	c.SetCookie("access_token", user.AccessToken, int(app.AccessTokenTTL), "/", "", false, true)
	c.SetCookie("refresh_token", user.RefreshToken, int(app.RefreshTokenTTL), "/", "", false, true)

	api.Success(c, user)
}

// @Summary Refresh User Token
// @Description Refresh user access token using refresh token
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/refresh-token [post]
func UserRefreshToken(c *gin.Context) {
	s := services.NewUserService()
	refreshToken, e := c.Cookie("refresh_token")
	if e != nil {
		api.Error(c, app.NewError(http.StatusUnauthorized, e.Error()))
		return
	}
	if refreshToken == "" {
		api.Error(c, app.NewError(http.StatusUnauthorized, "Refresh token is required"))
		return
	}

	user, err := s.RefreshToken(c, refreshToken)
	if err != nil {
		api.Error(c, err)
		return
	}

	c.SetCookie("access_token", user.AccessToken, int(app.AccessTokenTTL), "/", "", false, true)

	api.Success(c, user)
}

// @Summary Sign Out User
// @Description Sign out user by clearing cookies
// @Tags User
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param user body models.UserSignOutRequest true "User sign out request"
// @Success 200 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/{userID}/sign-out [post]
func UserSignOut(c *gin.Context) {
	var req models.UserSignOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewUserService()
	if err := s.SignOut(c, &req, refreshToken); err != nil {
		api.Error(c, err)
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	api.Success(c, nil)
}

// @Summary Update User
// @Description Update user information
// @Tags User
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param user body models.UserUpdateRequest true "User update request"
// @Success 200 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/{userID} [put]
func UserUpdate(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "User ID is required"))
		return
	}

	var req models.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.UserID = userID

	s := services.NewUserService()
	user, err := s.Update(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, user)
}

// @Summary Change User Role
// @Description Change user role
// @Tags User
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param user body models.UserChangeRoleRequest true "User change role request"
// @Success 200 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/{userID}/change-role [put]
func UserChangeRole(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "User ID is required"))
		return
	}

	var req models.UserChangeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.UserID = userID

	s := services.NewUserService()
	user, err := s.ChangeRole(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, user)
}

// @Summary Reset User Password
// @Description Reset user password
// @Tags User
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param user body models.UserResetPasswordRequest true "User reset password request"
// @Success 200 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/{userID}/reset-password [put]
func UserResetPassword(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "User ID is required"))
		return
	}

	var req models.UserResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.UserID = userID

	s := services.NewUserService()
	um, err := s.ResetPassword(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	api.Success(c, um)
}

// @Summary Rename User
// @Description Rename user with password verification
// @Tags User
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param user body models.UserRenameRequest true "User rename request"
// @Success 200 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/{userID}/rename [put]
func UserRename(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "User ID is required"))
		return
	}

	var req models.UserRenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.UserID = userID

	s := services.NewUserService()
	user, err := s.Rename(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, user)
}

// @Summary Delete User
// @Description Delete user by user ID
// @Tags User
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param user body models.DeleteUserRequest true "Delete user request"
// @Success 204 {object} api.Response{data=models.UserModel}
// @Router /api/v1/users/{userID} [delete]
func DeleteUser(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "User ID is required"))
		return
	}

	var req models.DeleteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.UserID = userID

	s := services.NewUserService()
	err := s.Delete(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	api.NoContent(c)
}

// @Summary Get all clusters and cluster nodes for admin
// @Description Get all clusters and cluster nodes for admin
// @Tags admin
// @Success 200 {object} api.Response{data=models.GetAdminResourcesResponse}
// @Router /api/v1/admin/resources [get]
func GetAdminResources(c *gin.Context) {
	svc := services.NewUserService()
	result, err := svc.GetAdminResources(c)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, result)
}

// @Summary Get all resources user can access
// @Description Get all ProjectRef, EnvRef, AppRef user has permission to query
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=models.GetUserResourcesResponse}
// @Router /api/v1/users/resources [get]
func GetUserResources(c *gin.Context) {
	svc := services.NewUserService()
	result, err := svc.GetUserResources(c)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, result)
}
