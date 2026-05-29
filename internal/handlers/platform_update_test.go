package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func platformUpdateClaimsMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID:   "u-admin",
			Username: "admin",
			Role:     role,
		})
		c.Next()
	}
}

func TestGetPlatformUpdateConfigReturnsDefaults(t *testing.T) {
	original := getPlatformUpdateConfig
	t.Cleanup(func() { getPlatformUpdateConfig = original })

	getPlatformUpdateConfig = func() (*models.PlatformUpdateConfig, error) {
		return &models.PlatformUpdateConfig{
			API: models.PlatformUpdateTargetConfig{
				ImageRepository: "ghcr.io/ketches/ketches/ketches-api",
				Namespace:       "ketches",
				DeploymentName:  "ketches-api",
				ContainerName:   "ketches-api",
			},
			UI: models.PlatformUpdateTargetConfig{
				ImageRepository: "ghcr.io/ketches/ketches/ketches-ui",
				Namespace:       "ketches",
				DeploymentName:  "ketches-ui",
				ContainerName:   "ketches-ui",
			},
		}, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-update")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleAdmin), middlewares.AdminOnly())
	group.GET("/config", GetPlatformUpdateConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform-update/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data models.PlatformUpdateConfig `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ghcr.io/ketches/ketches/ketches-api", resp.Data.API.ImageRepository)
	assert.Equal(t, "ghcr.io/ketches/ketches/ketches-ui", resp.Data.UI.ImageRepository)
}

func TestGetPlatformUpdateConfigRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-update")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleUser), middlewares.AdminOnly())
	group.GET("/config", GetPlatformUpdateConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform-update/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdatePlatformUpdateConfigValidatesRequiredTargets(t *testing.T) {
	original := updatePlatformUpdateConfig
	t.Cleanup(func() { updatePlatformUpdateConfig = original })

	called := false
	updatePlatformUpdateConfig = func(*models.PlatformUpdateConfig, *app.Claims) (*models.PlatformUpdateConfig, error) {
		called = true
		return nil, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-update")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleAdmin), middlewares.AdminOnly())
	group.PUT("/config", UpdatePlatformUpdateConfig)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/platform-update/config", strings.NewReader(`{"api":{"namespace":"ketches"}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, called)
}

func TestGetPlatformUpdateStatusReturnsBlockedRolloutState(t *testing.T) {
	original := getPlatformUpdateStatus
	t.Cleanup(func() { getPlatformUpdateStatus = original })

	getPlatformUpdateStatus = func() (*models.PlatformUpdateStatus, error) {
		return &models.PlatformUpdateStatus{
			LocalPlatformVersion: "v1.0.0",
			RunningInKubernetes:  false,
			CanRollout:           false,
			RolloutBlockers:      []string{"current platform is not running in kubernetes"},
		}, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-update")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleAdmin), middlewares.AdminOnly())
	group.GET("/status", GetPlatformUpdateStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform-update/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data models.PlatformUpdateStatus `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Data.CanRollout)
	assert.Equal(t, []string{"current platform is not running in kubernetes"}, resp.Data.RolloutBlockers)
}

func TestTriggerPlatformRolloutReturnsAccepted(t *testing.T) {
	original := triggerPlatformRollout
	t.Cleanup(func() { triggerPlatformRollout = original })

	triggerPlatformRollout = func(req *models.TriggerPlatformRolloutRequest, claims *app.Claims) (*models.PlatformUpdateRolloutResult, error) {
		assert.Equal(t, "v1.2.0", req.SharedVersion)
		assert.Equal(t, "u-admin", claims.UserID)
		return &models.PlatformUpdateRolloutResult{
			API: models.PlatformUpdateRolloutTarget{
				Namespace:      "ketches",
				DeploymentName: "ketches-api",
				ContainerName:  "ketches-api",
				PreviousImage:  "ghcr.io/ketches/ketches/ketches-api:v1.0.0",
				TargetImage:    "ghcr.io/ketches/ketches/ketches-api:v1.2.0",
			},
			UI: models.PlatformUpdateRolloutTarget{
				Namespace:      "ketches",
				DeploymentName: "ketches-ui",
				ContainerName:  "ketches-ui",
				PreviousImage:  "ghcr.io/ketches/ketches/ketches-ui:v1.0.0",
				TargetImage:    "ghcr.io/ketches/ketches/ketches-ui:v1.2.0",
			},
		}, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-update")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleAdmin), middlewares.AdminOnly())
	group.POST("/rollout", TriggerPlatformRollout)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform-update/rollout", strings.NewReader(`{"shared_version":"v1.2.0"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
	var resp struct {
		Data models.PlatformUpdateRolloutResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ghcr.io/ketches/ketches/ketches-api:v1.2.0", resp.Data.API.TargetImage)
	assert.Equal(t, "ghcr.io/ketches/ketches/ketches-ui:v1.2.0", resp.Data.UI.TargetImage)
}

func TestCheckPlatformUpdateReturnsStatus(t *testing.T) {
	original := checkPlatformUpdate
	t.Cleanup(func() { checkPlatformUpdate = original })

	checkPlatformUpdate = func(req *models.CheckPlatformUpdateRequest, claims *app.Claims) (*models.PlatformUpdateStatus, error) {
		assert.Equal(t, "auto", req.Mode)
		assert.Equal(t, "u-admin", claims.UserID)
		return &models.PlatformUpdateStatus{
			RecommendedSharedVersion: "v1.2.0",
		}, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-update")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleAdmin), middlewares.AdminOnly())
	group.POST("/check", CheckPlatformUpdate)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform-update/check", strings.NewReader(`{"mode":"auto"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data models.PlatformUpdateStatus `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "v1.2.0", resp.Data.RecommendedSharedVersion)
}

func TestCheckPlatformUpdateRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-update")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleUser), middlewares.AdminOnly())
	group.POST("/check", CheckPlatformUpdate)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform-update/check", strings.NewReader(`{"mode":"auto"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}
