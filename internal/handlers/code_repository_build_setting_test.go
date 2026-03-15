package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCodeRepositoryBuildSettingHandlerTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.ContainerRegistry{},
		&entities.BuildSetting{},
	))

	db.DB = testDB
}

func seedHandlerBuildSettingRegistry(t *testing.T, registryID string) {
	t.Helper()

	registry := entities.ContainerRegistry{
		ID:        registryID,
		Name:      "Main Registry",
		Provider:  entities.RegistryProviderGHCR,
		Endpoint:  "ghcr.io",
		Scope:     entities.RegistryScopeProject,
		ProjectID: "project-1",
		Enabled:   true,
	}
	require.NoError(t, db.DB.Create(&registry).Error)
}

func TestCreateRepoBuildSetting_BindsPlatformsAndCacheFields(t *testing.T) {
	setupCodeRepositoryBuildSettingHandlerTestDB(t)
	seedHandlerBuildSettingRegistry(t, "registry-1")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/code-repositories/:repoID/build-settings", CreateRepoBuildSetting)

	reqBody := map[string]any{
		"name":                   "backend",
		"image_name":             "demo/backend",
		"registry_id":            "registry-1",
		"platforms":              "linux/amd64,linux/arm64",
		"registry_cache_enabled": false,
		"registry_cache_ref":     "ghcr.io/demo/backend:buildcache-setting-1",
		"build_arg_pairs": []map[string]string{
			{"key": "ZETA", "value": "last"},
			{"key": "ALPHA", "value": "first"},
		},
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/code-repositories/repo-1/build-settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data models.BuildSettingResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "linux/amd64,linux/arm64", resp.Data.Platforms)
	assert.False(t, resp.Data.RegistryCacheEnabled)
	assert.Equal(t, "ghcr.io/demo/backend:buildcache-setting-1", resp.Data.RegistryCacheRef)
	assert.Equal(t, []models.BuildArgPair{
		{Key: "ALPHA", Value: "first"},
		{Key: "ZETA", Value: "last"},
	}, resp.Data.BuildArgPairs)
}

func TestGetRepoBuildSetting_ReturnsPlatformsCacheAndArgPairs(t *testing.T) {
	setupCodeRepositoryBuildSettingHandlerTestDB(t)
	seedHandlerBuildSettingRegistry(t, "registry-1")

	repoID := "repo-1"
	setting := entities.BuildSetting{
		ID:                   "setting-1",
		Name:                 "backend",
		CodeRepositoryID:     &repoID,
		GitRef:               "main",
		DockerfilePath:       "Dockerfile",
		BuildContext:         ".",
		ImageName:            "demo/backend",
		RegistryID:           "registry-1",
		BuildArgs:            "ALPHA=first\nZETA=last",
		Platforms:            "linux/amd64,linux/arm64",
		RegistryCacheEnabled: boolPtr(false),
		RegistryCacheRef:     "ghcr.io/demo/backend:buildcache-setting-1",
	}
	require.NoError(t, db.DB.Select("*").Create(&setting).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/code-repositories/:repoID/build-settings/:settingID", GetRepoBuildSetting)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/code-repositories/repo-1/build-settings/setting-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data models.BuildSettingResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "linux/amd64,linux/arm64", resp.Data.Platforms)
	assert.False(t, resp.Data.RegistryCacheEnabled)
	assert.Equal(t, "ghcr.io/demo/backend:buildcache-setting-1", resp.Data.RegistryCacheRef)
	assert.Equal(t, []models.BuildArgPair{
		{Key: "ALPHA", Value: "first"},
		{Key: "ZETA", Value: "last"},
	}, resp.Data.BuildArgPairs)
}

func boolPtr(v bool) *bool {
	return &v
}
