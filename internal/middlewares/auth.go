package middlewares

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
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
		if tokenString == "" {
			tokenString, _ = c.Cookie(app.LegacyAuthCookieName)
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

func ForwardAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, _ := c.Cookie(app.AccessTokenCookieName)
		if tokenString == "" {
			tokenString, _ = c.Cookie(app.LegacyAuthCookieName)
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
