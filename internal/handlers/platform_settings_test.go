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

func TestGetPlatformBrandingReturnsDefaults(t *testing.T) {
	original := getPlatformBranding
	t.Cleanup(func() { getPlatformBranding = original })

	getPlatformBranding = func() (*models.PlatformBrandingResponse, error) {
		return &models.PlatformBrandingResponse{
			Name: "Ketches Admin",
		}, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-settings")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleAdmin), middlewares.AdminOnly())
	group.GET("/branding", GetPlatformBranding)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform-settings/branding", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data models.PlatformBrandingResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Ketches Admin", resp.Data.Name)
}

func TestUpdatePlatformBrandingAcceptsJSON(t *testing.T) {
	original := updatePlatformBranding
	t.Cleanup(func() { updatePlatformBranding = original })

	updatePlatformBranding = func(req *models.UpdatePlatformBrandingRequest, claims *app.Claims) (*models.PlatformBrandingResponse, error) {
		assert.Equal(t, "Custom Admin", req.Name)
		assert.Equal(t, "u-admin", claims.UserID)
		return &models.PlatformBrandingResponse{
			Name: req.Name,
		}, nil
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/platform-settings")
	group.Use(platformUpdateClaimsMiddleware(app.UserRoleAdmin), middlewares.AdminOnly())
	group.PUT("/branding", UpdatePlatformBranding)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/platform-settings/branding", strings.NewReader(`{"name":"Custom Admin"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
