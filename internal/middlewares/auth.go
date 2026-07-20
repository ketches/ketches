package middlewares

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/services"
)

func verifyToken(tokenString, expectedTokenType string) (*app.Claims, any, error) {
	if tokenString == "" {
		return nil, nil, errors.New("token is required")
	}

	claims, err := app.ParseToken(tokenString, expectedTokenType)
	if err != nil {
		return nil, nil, errors.New("invalid token")
	}

	user, err := services.GetUser(claims.UserID)
	if err != nil || user == nil {
		return nil, nil, errors.New("user not found")
	}
	if user.IsLocked {
		return nil, nil, services.ErrAccountLocked
	}

	return claims, user, nil
}

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString, _ = c.Cookie(app.AccessTokenCookieName)
		}

		claims, user, err := verifyToken(tokenString, app.TokenTypeAccess)
		if err != nil {
			api.Error(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		c.Set("claims", claims)
		c.Set("user", user)
		c.Next()
	}
}

func RequirePasswordChange() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := c.Get("user")
		if !ok {
			api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
			c.Abort()
			return
		}

		user, ok := currentUser.(*entities.User)
		if !ok || user == nil {
			api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
			c.Abort()
			return
		}

		if !user.MustChangePassword || passwordChangeRouteAllowed(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}

		api.Error(c, http.StatusForbidden, services.ErrPasswordChangeRequired)
		c.Abort()
	}
}

func passwordChangeRouteAllowed(method, path string) bool {
	switch {
	case method == http.MethodGet && path == "/api/v1/users/me":
		return true
	case method == http.MethodPatch && path == "/api/v1/users/me/password":
		return true
	case method == http.MethodPost && path == "/api/v1/users/logout":
		return true
	default:
		return false
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.Get("claims")
		if !ok {
			api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
			c.Abort()
			return
		}

		if claims.(*app.Claims).Role != app.UserRoleAdmin {
			api.Error(c, http.StatusForbidden, errors.New("forbidden"))
			c.Abort()
			return
		}
		c.Next()
	}
}
