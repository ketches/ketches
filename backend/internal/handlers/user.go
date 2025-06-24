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
// @Param query query model.ListUsersRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=model.ListUsersResponse}
// @Router /api/v1/users [get]
// @Security BearerAuth
func ListUsers(c *gin.Context) {
	var req models.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	us := services.NewUserService()
	resp, err := us.List(c, &req)
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
// @Success 200 {object} api.Response{data=model.UserModel}
// @Router /api/v1/users/{userID} [get]
func GetUser(c *gin.Context) {
	var req models.GetUserProfileRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	us := services.NewUserService()
	user, err := us.Get(c, &req)
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
// @Param user body model.UserSignUpRequest true "User sign up request"
// @Success 201 {object} api.Response{data=model.UserModel}
// @Router /api/v1/users/sign-up [post]
func UserSignUp(c *gin.Context) {
	var req models.UserSignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	us := services.NewUserService()
	user, err := us.SignUp(c, &req)
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
// @Param user body model.UserSignInRequest true "User sign in request"
// @Success 201 {object} api.Response{data=model.UserModel}
// @Router /api/v1/users/sign-in [post]
func UserSignIn(c *gin.Context) {
	var req models.UserSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	us := services.NewUserService()
	user, err := us.SignIn(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	c.SetCookie("access_token", user.AccessToken, int(app.AccessTokenTTL), "/", "", true, true)
	c.SetCookie("refresh_token", user.RefreshToken, int(app.RefreshTokenTTL), "/", "", true, true)

	api.Success(c, user)
}

// @Summary Refresh User Token
// @Description Refresh user access token using refresh token
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=model.UserModel}
// @Router /api/v1/users/refresh-token [post]
func UserRefreshToken(c *gin.Context) {
	us := services.NewUserService()
	refreshToken, e := c.Cookie("refresh_token")
	if e != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, e.Error()))
		return
	}
	if refreshToken == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Refresh token is required"))
		return
	}

	user, err := us.RefreshToken(c, refreshToken)
	if err != nil {
		api.Error(c, err)
		return
	}

	c.SetCookie("access_token", user.AccessToken, int(app.AccessTokenTTL), "/", "", true, true)

	api.Success(c, user)
}

// @Summary Sign Out User
// @Description Sign out user by clearing cookies
// @Tags User
// @Accept json
// @Produce json
// @Param user body model.UserSignOutRequest true "User sign out request"
// @Success 200 {object} api.Response{data=model.UserModel}
// @Router /api/v1/users/sign-out [post]
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

	us := services.NewUserService()
	if err := us.SignOut(c, &req, refreshToken); err != nil {
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
// @Param user body model.UserUpdateRequest true "User update request"
// @Success 200 {object} api.Response{data=model.UserModel}
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

	us := services.NewUserService()
	user, err := us.Update(c, &req)
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
// @Param user body model.UserChangeRoleRequest true "User change role request"
// @Success 200 {object} api.Response{data=model.UserModel}
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

	us := services.NewUserService()
	user, err := us.ChangeRole(c, &req)
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
// @Param user body model.UserResetPasswordRequest true "User reset password request"
// @Success 200 {object} api.Response{data=model.UserModel}
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

	us := services.NewUserService()
	um, err := us.ResetPassword(c, &req)
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
// @Param user body model.UserRenameRequest true "User rename request"
// @Success 200 {object} api.Response{data=model.UserModel}
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

	us := services.NewUserService()
	user, err := us.Rename(c, &req)
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
// @Param user body model.DeleteUserRequest true "Delete user request"
// @Success 204 {object} api.Response{data=model.UserModel}
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

	us := services.NewUserService()
	err := us.Delete(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	api.NoContent(c)
}
