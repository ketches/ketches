package middlewares

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
)

// projectRoleRank defines the hierarchy of project-level roles.
// Higher rank means more permissions.
var projectRoleRank = map[string]int{
	"owner":     3,
	"developer": 2,
	"viewer":    1,
}

// resolveProjectID attempts to extract the project ID from URL parameters.
// It checks :projectID, :envID, and :appID in order, resolving the latter
// two via DB lookups through the env and app tables.
func resolveProjectID(c *gin.Context) (string, bool) {
	// Direct project ID from URL
	if projectID := c.Param("projectID"); projectID != "" {
		return projectID, true
	}

	// Resolve via env ID
	if envID := c.Param("envID"); envID != "" {
		var env entities.Env
		if err := db.DB.Select("project_id").Where("id = ?", envID).First(&env).Error; err != nil {
			return "", false
		}
		return env.ProjectID, true
	}

	// Resolve via app ID (join apps → envs)
	if appID := c.Param("appID"); appID != "" {
		var env entities.Env
		if err := db.DB.Select("envs.project_id").
			Joins("JOIN apps ON apps.env_id = envs.id").
			Where("apps.id = ?", appID).
			First(&env).Error; err != nil {
			return "", false
		}
		return env.ProjectID, true
	}

	return "", false
}

// RequireProjectRole returns a middleware that enforces a minimum project role.
// Users with the "admin" system role bypass the check entirely.
// Non-admin users must be a project member with a role rank >= the minimum.
func RequireProjectRole(minRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := api.GetClaims(c)
		if claims == nil {
			api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
			c.Abort()
			return
		}

		// Admin system role bypasses project role check
		if claims.Role == "admin" {
			c.Next()
			return
		}

		projectID, ok := resolveProjectID(c)
		if !ok {
			api.Error(c, http.StatusBadRequest, errors.New("unable to resolve project"))
			c.Abort()
			return
		}

		// Look up the user's project membership
		var member entities.ProjectMember
		if err := db.DB.Where("project_id = ? AND user_id = ?", projectID, claims.UserID).
			First(&member).Error; err != nil {
			api.Error(c, http.StatusForbidden, errors.New("insufficient permissions"))
			c.Abort()
			return
		}

		// Compare role ranks
		if projectRoleRank[member.ProjectRole] < projectRoleRank[minRole] {
			api.Error(c, http.StatusForbidden, errors.New("insufficient permissions"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// BlockViewer is a convenience middleware that blocks users with the "viewer"
// project role. It requires at least "developer" level access.
func BlockViewer() gin.HandlerFunc {
	return RequireProjectRole("developer")
}
