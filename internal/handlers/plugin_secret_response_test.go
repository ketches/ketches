package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPluginHandlerSecretTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Plugin{}, &entities.AppPlugin{}))

	db.DB = testDB
}

func TestGetPluginDoesNotExposeRegistryPassword(t *testing.T) {
	setupPluginHandlerSecretTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Plugin{
		ID:               "plugin-1",
		ProjectID:        "project-1",
		Slug:             "plugin-one",
		Name:             "Plugin One",
		Image:            "docker.io/library/migrate:latest",
		RegistryUsername: "robot",
		RegistryPassword: "enc:v1:opaque",
		PluginType:       "init",
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/projects/:projectID/plugins/:pluginID", GetPlugin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/plugins/plugin-1", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "projectID", Value: "project-1"}, {Key: "pluginID", Value: "plugin-1"}}
	ctx.Set("claims", &app.Claims{Role: "admin"})
	ctx.Set("user", &entities.User{Base: entities.Base{ID: "user-1"}, Username: "demo", Role: "admin"})

	GetPlugin(ctx)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	_, exists := data["registry_password"]
	assert.False(t, exists)
	assert.Equal(t, true, data["has_registry_password"])

	_ = r
}

func TestListPluginsDoesNotExposeRegistryPassword(t *testing.T) {
	setupPluginHandlerSecretTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Plugin{
		ID:               "plugin-1",
		ProjectID:        "project-1",
		Slug:             "plugin-one",
		Name:             "Plugin One",
		Image:            "docker.io/library/migrate:latest",
		RegistryUsername: "robot",
		RegistryPassword: "enc:v1:opaque",
		PluginType:       "init",
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/projects/:projectID/plugins", ListPlugins)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/plugins?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "projectID", Value: "project-1"}}
	ctx.Set("claims", &app.Claims{Role: "admin"})
	ctx.Set("user", &entities.User{Base: entities.Base{ID: "user-1"}, Username: "demo", Role: "admin"})

	ListPlugins(ctx)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	_, exists := item["registry_password"]
	assert.False(t, exists)
	assert.Equal(t, true, item["has_registry_password"])

	_ = r
}

func TestCreatePluginResponseDoesNotExposeRegistryPassword(t *testing.T) {
	setupPluginHandlerSecretTestDB(t)

	resp := toPluginResponse(&entities.Plugin{
		ID:               "plugin-1",
		Slug:             "plugin-one",
		Name:             "Plugin One",
		Image:            "docker.io/library/migrate:latest",
		RegistryUsername: "robot",
		RegistryPassword: "enc:v1:opaque",
		PluginType:       "init",
	})

	body, err := json.Marshal(resp)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	_, exists := payload["registry_password"]
	assert.False(t, exists)
	assert.Equal(t, true, payload["has_registry_password"])
}
