package middlewares

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/services"
)

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
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			api.Error(c, http.StatusUnauthorized, jwt.ErrTokenSignatureInvalid)
			c.Abort()
			return
		}

		claims := &app.Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return []byte(app.Config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			api.Error(c, http.StatusUnauthorized, jwt.ErrTokenSignatureInvalid)
			c.Abort()
			return
		}

		user, err := services.GetUser(claims.UserID)
		if err != nil || user == nil {
			api.Error(c, http.StatusUnauthorized, errors.New("user not found"))
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
			api.Error(c, http.StatusUnauthorized, jwt.ErrTokenSignatureInvalid)
			c.Abort()
			return
		}

		if claims.(*app.Claims).Role != "admin" {
			api.Error(c, http.StatusForbidden, jwt.ErrTokenSignatureInvalid)
			c.Abort()
			return
		}
		c.Next()
	}
}
