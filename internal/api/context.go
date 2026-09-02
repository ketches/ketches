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

func SetProjectRole(c *gin.Context, role string) {
	c.Set("projectRole", role)
}

func GetProjectRole(c *gin.Context) app.ProjectRole {
	role, ok := c.Get("projectRole")
	if !ok {
		return ""
	}
	value, ok := role.(string)
	if !ok {
		return ""
	}
	switch value {
	case app.ProjectRoleOwner, app.ProjectRoleDeveloper, app.ProjectRoleViewer:
		return value
	default:
		return ""
	}
}
