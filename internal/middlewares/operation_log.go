package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/services"
)

const maxOperationLogBodySize = 4096
const redactedOperationLogValue = "[REDACTED]"

func OperationLog() gin.HandlerFunc {
	rules := operationLogRouteRules()
	return func(c *gin.Context) {
		if shouldSkipOperationLog(c) {
			c.Next()
			return
		}

		bodySummary, bodyAction, bodyUsername := captureRequestBody(c)
		c.Next()

		fullPath := c.FullPath()
		if fullPath == "" {
			return
		}
		rule, ok := rules[c.Request.Method+" "+fullPath]
		if !ok {
			return
		}

		action := rule.Action
		sensitivity := rule.Sensitivity
		if rule.BodyActionField != "" && bodyAction != "" {
			if mapped, exists := rule.BodyActionToAction[bodyAction]; exists {
				action = mapped
			}
			if mapped, exists := rule.BodyActionSensitive[bodyAction]; exists {
				sensitivity = mapped
			}
		}

		status := entities.OperationLogStatusSuccess
		if c.Writer.Status() >= http.StatusBadRequest {
			status = entities.OperationLogStatusFailure
		}

		claims := api.GetClaims(c)
		resourceID := c.Param(rule.ResourceIDParam)
		projectID, envID, appID, repoID := resolveOperationContextIDs(c)
		if rule.ResourceIDParam == "appID" && appID == "" {
			appID = resourceID
		}
		if rule.ResourceIDParam == "repoID" && repoID == "" {
			repoID = resourceID
		}

		input := services.CreateOperationLogInput{
			Action:         action,
			ResourceType:   rule.ResourceType,
			ResourceID:     resourceID,
			Status:         status,
			StatusCode:     c.Writer.Status(),
			Sensitivity:    sensitivity,
			RequestSummary: bodySummary,
			ClientIP:       c.ClientIP(),
			ProjectID:      projectID,
			EnvID:          envID,
			AppID:          appID,
			RepoID:         repoID,
		}
		if claims != nil {
			input.UserID = claims.UserID
			input.Username = claims.Username
		} else if bodyUsername != "" {
			input.Username = bodyUsername
		}
		if input.Username == "" {
			input.Username = "anonymous"
		}
		_ = services.CreateOperationLog(input)
	}
}

func shouldSkipOperationLog(c *gin.Context) bool {
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
		return false
	}
	path := c.Request.URL.Path
	return strings.Contains(path, "/logs") ||
		strings.Contains(path, "/exec") ||
		strings.Contains(path, "/proxy/") ||
		strings.Contains(path, "/upload") ||
		strings.Contains(path, "/download") ||
		strings.Contains(path, "/files")
}

func captureRequestBody(c *gin.Context) (string, string, string) {
	if c.Request.Body == nil {
		return "", "", ""
	}
	contentType := strings.ToLower(strings.TrimSpace(c.ContentType()))
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxOperationLogBodySize+1))
	if err != nil {
		return "", "", ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(body) == 0 {
		return "", "", ""
	}
	if len(body) > maxOperationLogBodySize {
		body = body[:maxOperationLogBodySize]
	}
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return "[multipart form data omitted]", "", ""
	}
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		return "[form payload omitted]", "", ""
	}

	bodyAction := ""
	bodyUsername := ""
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		if value, ok := payload["action"].(string); ok {
			bodyAction = value
		}
		if value, ok := payload["username"].(string); ok {
			bodyUsername = strings.TrimSpace(value)
		}

		sanitized, err := json.Marshal(sanitizeOperationLogValue(payload))
		if err == nil {
			return strings.TrimSpace(string(sanitized)), bodyAction, bodyUsername
		}
		return "[json payload omitted]", bodyAction, bodyUsername
	}
	if strings.HasPrefix(contentType, "application/json") {
		return "[json payload omitted]", bodyAction, bodyUsername
	}

	return strings.TrimSpace(string(body)), bodyAction, bodyUsername
}

func sanitizeOperationLogValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, entry := range typed {
			if isSensitiveOperationLogField(key) {
				sanitized[key] = redactedOperationLogValue
				continue
			}
			sanitized[key] = sanitizeOperationLogValue(entry)
		}
		return sanitized
	case []any:
		sanitized := make([]any, len(typed))
		for i := range typed {
			sanitized[i] = sanitizeOperationLogValue(typed[i])
		}
		return sanitized
	default:
		return value
	}
}

func isSensitiveOperationLogField(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case
		"password",
		"current_password",
		"new_password",
		"token",
		"access_token",
		"refresh_token",
		"api_key",
		"secret",
		"secret_key",
		"private_key",
		"kube_config",
		"kubeconfig",
		"git_password",
		"registry_password",
		"ca_cert",
		"cert",
		"key":
		return true
	default:
		return false
	}
}

func resolveOperationContextIDs(c *gin.Context) (string, string, string, string) {
	projectID := c.Param("projectID")
	envID := c.Param("envID")
	appID := c.Param("appID")
	repoID := c.Param("repoID")

	if appID != "" {
		appCtx, err := services.GetAppContext(c.Request.Context(), appID)
		if err == nil && appCtx != nil {
			envID = appCtx.EnvContext.Env.ID
			projectID = appCtx.EnvContext.Project.ID
			if appCtx.App.CodeRepositoryID != nil {
				repoID = *appCtx.App.CodeRepositoryID
			}
		}
	}
	if envID != "" && projectID == "" {
		env, err := services.GetEnv(envID)
		if err == nil && env != nil {
			projectID = env.ProjectID
		}
	}
	if repoID != "" && projectID == "" {
		repo, err := services.GetCodeRepository(repoID)
		if err == nil && repo != nil {
			projectID = repo.ProjectID
		}
	}
	return projectID, envID, appID, repoID
}
