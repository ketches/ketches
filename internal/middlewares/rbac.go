package middlewares

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type batchDeleteAppsRequest struct {
	IDs []string `json:"ids"`
}

type appProjectRow struct {
	ID        string
	ProjectID string
}

func resolveBatchDeleteProjectID(c *gin.Context) (string, bool) {
	if c.Request.Method != http.MethodPost || c.FullPath() != "/api/v1/apps/batch-delete" {
		return "", false
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Debug("resolveProjectID batch delete body read failed", "error", err)
		return "", false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var req batchDeleteAppsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		slog.Debug("resolveProjectID batch delete body decode failed", "error", err)
		return "", false
	}
	if len(req.IDs) == 0 {
		return "", false
	}

	uniqueIDs := make(map[string]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		if id == "" {
			return "", false
		}
		uniqueIDs[id] = struct{}{}
	}

	appIDs := make([]string, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		appIDs = append(appIDs, id)
	}

	var rows []appProjectRow
	if err := db.DB.Table("apps").
		Select("apps.id, envs.project_id").
		Joins("JOIN envs ON envs.id = apps.env_id").
		Where("apps.id IN ?", appIDs).
		Scan(&rows).Error; err != nil {
		slog.Debug("resolveProjectID batch delete lookup failed", "error", err)
		return "", false
	}
	if len(rows) != len(appIDs) {
		return "", false
	}

	projectID := rows[0].ProjectID
	if projectID == "" {
		return "", false
	}
	for _, row := range rows[1:] {
		if row.ProjectID != projectID {
			return "", false
		}
	}

	return projectID, true
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

	// Resolve body-driven batch app routes that do not carry project context in the path.
	if projectID, ok := resolveBatchDeleteProjectID(c); ok {
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

// RequireRecycleBinOwner protects body-driven recycle-bin actions. Since these
// routes do not carry a project ID in the path, resolve every requested
// resource to its project and require owner membership for each one. The
// service layer repeats this check inside its transaction.
func RequireRecycleBinOwner(resource string) gin.HandlerFunc {
	validResources := map[string]bool{
		"apps":              true,
		"envs":              true,
		"projects":          true,
		"code-repositories": true,
	}
	if !validResources[resource] {
		panic(fmt.Sprintf("RequireRecycleBinOwner: unknown resource %q", resource))
	}

	return func(c *gin.Context) {
		claims := api.GetClaims(c)
		if claims == nil {
			api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
			c.Abort()
			return
		}
		if claims.Role == app.UserRoleAdmin {
			c.Next()
			return
		}

		ids, err := readRecycleBinIDs(c)
		if err != nil {
			api.Error(c, http.StatusBadRequest, err)
			c.Abort()
			return
		}
		projectIDs, err := resolveRecycleBinProjectIDs(resource, ids)
		if err != nil {
			api.Error(c, http.StatusNotFound, err)
			c.Abort()
			return
		}
		for _, projectID := range projectIDs {
			var count int64
			if err := db.DB.Model(&entities.ProjectMember{}).
				Where("project_id = ? AND user_id = ? AND project_role = ?", projectID, claims.UserID, app.ProjectRoleOwner).
				Count(&count).Error; err != nil {
				api.Error(c, http.StatusInternalServerError, err)
				c.Abort()
				return
			}
			if count == 0 {
				api.Error(c, http.StatusForbidden, errors.New("insufficient permissions"))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func readRecycleBinIDs(c *gin.Context) ([]string, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var request struct {
		IDs []string `json:"ids"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, err
	}
	if len(request.IDs) == 0 {
		return nil, errors.New("resource IDs are required")
	}

	unique := make([]string, 0, len(request.IDs))
	seen := make(map[string]struct{}, len(request.IDs))
	for _, id := range request.IDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, errors.New("resource IDs must not be empty")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique, nil
}

func resolveRecycleBinProjectIDs(resource string, ids []string) ([]string, error) {
	projectIDs := make([]string, 0, len(ids))
	switch resource {
	case "projects":
		var rows []struct{ ID string }
		if err := db.DB.Unscoped().Table("projects").Select("id").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
			return nil, err
		}
		if len(rows) != len(ids) {
			return nil, errors.New("recycle-bin project not found")
		}
		for _, row := range rows {
			projectIDs = append(projectIDs, row.ID)
		}
	case "envs":
		var rows []struct {
			ProjectID string `gorm:"column:project_id"`
		}
		if err := db.DB.Unscoped().Table("envs").Select("project_id").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
			return nil, err
		}
		if len(rows) != len(ids) {
			return nil, errors.New("recycle-bin environment not found")
		}
		for _, row := range rows {
			projectIDs = append(projectIDs, row.ProjectID)
		}
	case "apps":
		var rows []struct {
			ProjectID string `gorm:"column:project_id"`
		}
		if err := db.DB.Unscoped().Table("apps").Select("envs.project_id").
			Joins("JOIN envs ON envs.id = apps.env_id").Where("apps.id IN ?", ids).Scan(&rows).Error; err != nil {
			return nil, err
		}
		if len(rows) != len(ids) {
			return nil, errors.New("recycle-bin application not found")
		}
		for _, row := range rows {
			projectIDs = append(projectIDs, row.ProjectID)
		}
	case "code-repositories":
		var rows []struct {
			ProjectID string `gorm:"column:project_id"`
		}
		if err := db.DB.Unscoped().Table("code_repositories").Select("project_id").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
			return nil, err
		}
		if len(rows) != len(ids) {
			return nil, errors.New("recycle-bin code repository not found")
		}
		for _, row := range rows {
			projectIDs = append(projectIDs, row.ProjectID)
		}
	}
	return projectIDs, nil
}

// BlockViewer is a convenience middleware that blocks users with the "viewer"
// project role. It requires at least "developer" level access.
func BlockViewer() gin.HandlerFunc {
	return RequireProjectRole(app.ProjectRoleDeveloper)
}
