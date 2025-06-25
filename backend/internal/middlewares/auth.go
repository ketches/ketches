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

package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := c.Cookie("access_token")
		if err != nil {
			auth := c.GetHeader("Authorization")
			if auth != "" {
				parts := strings.Split(auth, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					accessToken = parts[1]
				}
			}
		}

		if accessToken == "" {
			api.Error(c, app.NewError(http.StatusUnauthorized, "Access token is required"))
			return
		}

		claims, err := app.ValidateToken(accessToken)
		if err != nil {
			api.Error(c, app.NewError(http.StatusUnauthorized, err.Error()))
			return
		}

		user := new(entities.User)
		if err := db.Instance().Select("id, role").First(user, "id = ?", claims.UserID).Error; err != nil {
			if db.IsErrRecordNotFound(err) {
				api.Error(c, app.NewError(http.StatusNotFound, "User not found"))
				return
			}
			api.Error(c, app.NewError(http.StatusInternalServerError, "Database error: "+err.Error()))
			return
		}

		api.SetUserID(c, user.ID)
		api.SetUserRole(c, user.Role)

		// TODO: check user must reset password
		if user.MustResetPassword {
			api.Error(c, app.ErrUserPasswordMustReset)
			c.Abort()
			return
		}

		c.Next()
	}
}
