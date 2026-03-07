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

func verifyToken(tokenString string) (*app.Claims, any, error) {
	if tokenString == "" {
		return nil, nil, jwt.ErrTokenSignatureInvalid
	}

	claims := &app.Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(app.Config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, nil, jwt.ErrTokenSignatureInvalid
	}

	user, err := services.GetUser(claims.UserID)
	if err != nil || user == nil {
		return nil, nil, errors.New("user not found")
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
			tokenString = c.Query("token")
		}

		claims, user, err := verifyToken(tokenString)
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
		// X-Ketches-Token cookie is used by the gateway quick-access feature so
		// the JWT never appears in the browser address bar.
		tokenString, _ := c.Cookie("X-Ketches-Token")

		claims, user, err := verifyToken(tokenString)
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
			api.Error(c, http.StatusUnauthorized, jwt.ErrTokenSignatureInvalid)
			c.Abort()
			return
		}

		if claims.(*app.Claims).Role != app.UserRoleAdmin {
			api.Error(c, http.StatusForbidden, jwt.ErrTokenSignatureInvalid)
			c.Abort()
			return
		}
		c.Next()
	}
}
