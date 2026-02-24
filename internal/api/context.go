package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
)

func GetClaims(c *gin.Context) *app.Claims {
	claims, ok := c.Get("claims")
	if !ok {
		return nil
	}
	return claims.(*app.Claims)
}

func GetUserRole(c *gin.Context) string {
	claims := GetClaims(c)
	if claims == nil {
		return ""
	}
	return claims.Role
}
