package middlewares

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/services"
)

func BuilderPreviewAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(services.BuilderPreviewSessionCookieName)
		if err != nil {
			api.Error(c, http.StatusUnauthorized, errors.New("builder preview session cookie is required"))
			c.Abort()
			return
		}

		claims, err := services.ParseBuilderPreviewSessionToken(tokenString)
		if err != nil {
			api.Error(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}
		if claims.ProjectID != c.Param("projectID") || claims.SessionID != c.Param("sessionID") || claims.RunID != c.Param("runID") {
			api.Error(c, http.StatusUnauthorized, errors.New("builder preview session scope mismatch"))
			c.Abort()
			return
		}
		user, err := services.GetUser(claims.UserID)
		if err != nil || user == nil {
			api.Error(c, http.StatusUnauthorized, errors.New("user not found"))
			c.Abort()
			return
		}

		c.Set("builderPreviewClaims", claims)
		c.Set("claims", &app.Claims{UserID: claims.UserID, Role: user.Role})
		c.Set("user", user)
		c.Next()
	}
}
