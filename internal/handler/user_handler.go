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

package handler

import (
	"github.com/ketches/ketches/internal/http"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/model"
	"github.com/ketches/ketches/internal/service"
	"github.com/ketches/ketches/util/jwt"
)

func ListUsers(c *gin.Context) {
	var g model.SpaceUri
	if err := c.BindUri(&g); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	var uf model.UserFilter
	if err := c.ShouldBindQuery(&uf); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	uf.SpaceUri = g

	us := service.NewUserService()
	users, err := us.List(c, &uf)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, users)
}

func GetUser(c *gin.Context) {
	accountID := c.Param("account_id")

	us := service.NewUserService()
	user, err := us.Get(c, accountID)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, user)
}

func UserSignUp(c *gin.Context) {
	var um model.UserSignUpModel
	if err := c.ShouldBindJSON(&um); err != nil {
		http.BadRequestWithError(c, err)
		return
	}

	us := service.NewUserService()
	user, err := us.SignUp(c, &um)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, user)
}

func UserSignIn(c *gin.Context) {
	var um model.UserSignInRequest
	if err := c.ShouldBindJSON(&um); err != nil {
		http.BadRequestWithError(c, err)
		return
	}

	us := service.NewUserService()
	user, err := us.SignIn(c, &um)
	if err != nil {
		http.Error(c, err)
		return
	}

	// Generate access token
	accessToken, err := jwt.GenerateToken(jwt.MyClaims{
		AccountID: user.AccountID,
		Email:     user.Email,
		TokenType: jwt.TokenTypeAccessToken,
	})
	if err != nil {
		http.Error(c, err)
		return
	}

	// Generate refresh token
	refreshToken, err := jwt.GenerateToken(jwt.MyClaims{
		AccountID: user.AccountID,
		Email:     user.Email,
		TokenType: jwt.TokenTypeRefreshToken,
	})
	if err != nil {
		log.Println("Generate refresh token failed:", err)
		return
	}

	// c.SetCookie("access_token", accessToken, 3600*2, "/", "", false, true)
	// c.SetCookie("refresh_token", refreshToken, 3600*24*7, "/", "", false, true)

	// api.SetSessions(c, map[string]any{
	// 	"access_token":  accessToken,
	// 	"refresh_token": refreshToken,
	// })

	// session := sessions.Default(c)
	// session.Options(sessions.Options{
	// 	SameSite: http.SameSiteNoneMode,
	// 	Secure:   true,
	// 	HttpOnly: true,
	// })
	// session.Set("account_id", user.AccountID)
	// session.Save()

	user.AccessToken = accessToken
	user.RefreshToken = refreshToken

	http.Success(c, user)
}

// UserRefreshToken refreshes access token
func UserRefreshToken(c *gin.Context) {
	var req model.UserRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}

	us := service.NewUserService()
	user, err := us.RefreshToken(c, req.RefreshToken)
	if err != nil {
		http.UnauthorizedWithError(c, err)
		return
	}

	c.SetCookie("access_token", user.AccessToken, 3600*2, "/", "", true, true)

	// api.NewSessioner(c).SetSessions(map[string]any{
	// 	"access_token":  user.AccessToken,
	// 	"refresh_token": user.RefreshToken,
	// })

	http.Success(c, user)
}

func UserSignOut(c *gin.Context) {
	var req model.UserSignOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	// api.DeleteSessions(c, []string{"access_token", "refresh_token"})

	// session := sessions.Default(c)
	// session.Delete("account_id")
	// session.Save()

	http.Success(c, nil)
}

func UserUpdate(c *gin.Context) {
	var g model.UserUri
	if err := c.BindUri(&g); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.UserUri = g

	us := service.NewUserService()
	user, err := us.Update(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, user)
}

func UserResetPassword(c *gin.Context) {
	var g model.UserUri
	if err := c.BindUri(&g); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	var req model.UserResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.UserUri = g

	us := service.NewUserService()
	um, err := us.ResetPassword(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	http.Success(c, um)
}

func UserRename(c *gin.Context) {
	var g model.UserUri
	if err := c.BindUri(&g); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	var req model.UserRenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}

	req.UserUri = g

	us := service.NewUserService()
	user, err := us.Rename(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}

	http.Success(c, user)
}

func DeleteUser(c *gin.Context) {
	var g model.UserUri
	if err := c.BindUri(&g); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	var req model.DeleteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		http.BadRequestWithError(c, err)
		return
	}
	req.UserUri = g

	us := service.NewUserService()
	err := us.Delete(c, &req)
	if err != nil {
		http.Error(c, err)
		return
	}
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	// session := sessions.Default(c)
	// session.Delete("access_token")
	// session.Delete("refresh_token")
	// session.Save()

	// session := sessions.Default(c)
	// session.Clear()
	// session.Save()

	http.Success(c, nil)
}
