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

package middleware

import (
	"errors"
	"github.com/ketches/ketches/internal/http"
	"github.com/ketches/ketches/pkg/ketches"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ketches/ketches/internal/global"
	jwtutil "github.com/ketches/ketches/util/jwt"
)

var (
	MsgUserPasswordMustReset = "user password must be reset"
	ErrUserPasswordMustReset = errors.New(MsgUserPasswordMustReset)
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		skipPaths := []string{
			"/api/v1/users/sign-in",
			"/api/v1/users/sign-up",
			"/api/v1/users/refresh-token",
			"/api/v1/users/reset-password",
		}
		if slices.Contains(skipPaths, c.Request.URL.Path) {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			http.AbortWithUnauthorized(c)
			return
		}

		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.AbortWithUnauthorized(c)
			return
		}
		accessToken := parts[1]
		claims, err := jwtutil.ValidateToken(accessToken)
		if err != nil {
			if claims != nil && claims.TokenType == jwtutil.TokenTypeAccessToken && errors.Is(err, jwt.ErrTokenExpired) {
				http.AbortWithAccessTokenExpired(c)
				return
			}
			http.AbortWithInvalidAccessToken(c)
			return
		}

		user, err := ketches.Store().UserLister().Get(claims.AccountID)
		if err != nil {
			http.ForbiddenWithError(c, err)
			c.Abort()
			return
		}

		// Current user must reset password
		if user.Spec.MustResetPassword {
			http.ForbiddenWithError(c, ErrUserPasswordMustReset)
			c.Abort()
			return
		}

		c.Set(global.ContextKeyAccountID, claims.AccountID)
		// 	登入角色，只有管理员可以切换 admin 和 user 角色登入，其他用户只能以 user 角色登入
		if claims.SignInRole != jwtutil.SignInRoleAdmin {
			claims.SignInRole = jwtutil.SignInRoleUser
		}
		c.Set(global.ContextKeySignInRole, claims.SignInRole)
		c.Set(global.ContextKeyEmail, claims.Email)

		c.Next()
	}
}
