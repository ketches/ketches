package middlewares

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
)

// projectRoleRank defines the hierarchy of project-level roles.
// Higher rank means more permissions.
var projectRoleRank = map[string]int{
	app.ProjectRoleOwner:     3,
	app.ProjectRoleDeveloper: 2,
	app.ProjectRoleViewer:    1,
}

// resolveProjectID attempts to extract the project ID from URL parameters.
// It checks :projectID, :envID, and :appID in order, resolving the latter
// two via DB lookups through the env and app tables.
func resolveProjectID(c *gin.Context) (string, bool) {
	// Guard against uninitialized DB (startup race / misconfiguration).
	if db.DB == nil {
		return "", false
	}

	// Direct project ID from URL
	if projectID := c.Param("projectID"); projectID != "" {
		return projectID, true
	}

	// Resolve via env ID
	if envID := c.Param("envID"); envID != "" {
		var env entities.Env
		if err := db.DB.Select("project_id").Where("id = ?", envID).First(&env).Error; err != nil {
			slog.Debug("resolveProjectID env lookup failed", "envID", envID, "error", err)
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
			slog.Debug("resolveProjectID app lookup failed", "appID", appID, "error", err)
			return "", false
		}
		return env.ProjectID, true
	}

	// Resolve flat app sub-resources via :id param (env-vars, volumes, config-files, gateways)
	// Use the route path prefix to determine which table to look up.
	if resourceID := c.Param("id"); resourceID != "" {
		path := c.FullPath()
		var appID string

		switch {
		case strings.HasPrefix(path, "/api/v1/env-vars/"):
			var r entities.AppEnvVar
			if err := db.DB.Select("app_id").Where("id = ?", resourceID).First(&r).Error; err != nil {
				slog.Debug("resolveProjectID env var lookup failed", "resourceID", resourceID, "error", err)
				return "", false
			}
			appID = r.AppID
		case strings.HasPrefix(path, "/api/v1/volumes/"):
			var r entities.AppVolume
			if err := db.DB.Select("app_id").Where("id = ?", resourceID).First(&r).Error; err != nil {
				slog.Debug("resolveProjectID volume lookup failed", "resourceID", resourceID, "error", err)
				return "", false
			}
			appID = r.AppID
		case strings.HasPrefix(path, "/api/v1/config-files/"):
			var r entities.AppConfigFile
			if err := db.DB.Select("app_id").Where("id = ?", resourceID).First(&r).Error; err != nil {
				slog.Debug("resolveProjectID config file lookup failed", "resourceID", resourceID, "error", err)
				return "", false
			}
			appID = r.AppID
		case strings.HasPrefix(path, "/api/v1/gateways/"):
			var r entities.AppGateway
			if err := db.DB.Select("app_id").Where("id = ?", resourceID).First(&r).Error; err != nil {
				slog.Debug("resolveProjectID gateway lookup failed", "resourceID", resourceID, "error", err)
				return "", false
			}
			appID = r.AppID
		default:
			return "", false
		}

		// Resolve appID → projectID via apps → envs join
		var env entities.Env
		if err := db.DB.Select("envs.project_id").
			Joins("JOIN apps ON apps.env_id = envs.id").
			Where("apps.id = ?", appID).
			First(&env).Error; err != nil {
			slog.Debug("resolveProjectID flat resource app lookup failed", "appID", appID, "error", err)
			return "", false
		}
		return env.ProjectID, true
	}

	if gatewayID := c.Param("gatewayID"); gatewayID != "" {
		var gateway entities.AppGateway
		if err := db.DB.Select("app_id").Where("id = ?", gatewayID).First(&gateway).Error; err != nil {
			slog.Debug("resolveProjectID gatewayID lookup failed", "gatewayID", gatewayID, "error", err)
			return "", false
		}

		var env entities.Env
		if err := db.DB.Select("envs.project_id").
			Joins("JOIN apps ON apps.env_id = envs.id").
			Where("apps.id = ?", gateway.AppID).
			First(&env).Error; err != nil {
			slog.Debug("resolveProjectID gateway env lookup failed", "gatewayID", gatewayID, "error", err)
			return "", false
		}
		return env.ProjectID, true
	}

	// Resolve via container registry ID
	if registryID := c.Param("registryID"); registryID != "" {
		var registry entities.ContainerRegistry
		if err := db.DB.Select("project_id").Where("id = ?", registryID).First(&registry).Error; err != nil {
			slog.Debug("resolveProjectID registry lookup failed", "registryID", registryID, "error", err)
			return "", false
		}
		// Cluster-scoped registries have no project context — skip RBAC check
		if registry.ProjectID == "" {
			return "", false
		}
		return registry.ProjectID, true
	}

	// Resolve via repo ID
	if repoID := c.Param("repoID"); repoID != "" {
		var repo entities.CodeRepository
		if err := db.DB.Select("project_id").Where("id = ?", repoID).First(&repo).Error; err != nil {
			slog.Debug("resolveProjectID repository lookup failed", "repoID", repoID, "error", err)
			return "", false
		}
		return repo.ProjectID, true
	}

	// Resolve via build setting ID
	if settingID := c.Param("settingID"); settingID != "" {
		var repo entities.CodeRepository
		if err := db.DB.Select("code_repositories.project_id").
			Table("build_settings").
			Joins("JOIN code_repositories ON code_repositories.id = build_settings.code_repository_id").
			Where("build_settings.id = ?", settingID).
			First(&repo).Error; err != nil {
			slog.Debug("resolveProjectID build setting lookup failed", "settingID", settingID, "error", err)
			return "", false
		}
		return repo.ProjectID, true
	}

	// Resolve via app group ID → env → project
	if groupID := c.Param("groupID"); groupID != "" {
		var env entities.Env
		if err := db.DB.Select("envs.project_id").
			Joins("JOIN app_groups ON app_groups.env_id = envs.id").
			Where("app_groups.id = ?", groupID).
			First(&env).Error; err != nil {
			slog.Debug("resolveProjectID app group lookup failed", "groupID", groupID, "error", err)
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
	// Panic at setup time for unknown roles — catches misconfiguration before any request.
	if _, ok := projectRoleRank[minRole]; !ok {
		panic(fmt.Sprintf("RequireProjectRole: unknown role %q", minRole))
	}

	return func(c *gin.Context) {
		claims := api.GetClaims(c)
		if claims == nil {
			api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
			c.Abort()
			return
		}

		// Admin system role bypasses project role check
		if claims.Role == app.UserRoleAdmin {
			c.Next()
			return
		}

		projectID, ok := resolveProjectID(c)
		if !ok {
			api.Error(c, http.StatusBadRequest, errors.New("unable to resolve project"))
			c.Abort()
			return
		}

		// Guard against uninitialized DB before member lookup
		if db.DB == nil {
			api.Error(c, http.StatusInternalServerError, errors.New("service unavailable"))
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
	return RequireProjectRole(app.ProjectRoleDeveloper)
}
