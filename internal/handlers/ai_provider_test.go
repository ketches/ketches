package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAIProviderHandlerTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)

	db.DB = testDB
	require.NoError(t, db.Migrate())
	require.NoError(t, db.DB.Create(&entities.User{Base: entities.Base{ID: "user-1"}, Username: "user", Email: "user@example.com", Password: "pw", Role: app.UserRoleUser}).Error)
	require.NoError(t, db.DB.Create(&entities.User{Base: entities.Base{ID: "owner-1"}, Username: "owner", Email: "owner@example.com", Password: "pw", Role: app.UserRoleUser}).Error)
	require.NoError(t, db.DB.Create(&entities.Project{Base: entities.Base{ID: "project-1"}, Slug: "demo-project", Name: "Demo Project"}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{ID: "member-owner", ProjectID: "project-1", UserID: "owner-1", ProjectRole: app.ProjectRoleOwner}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{ID: "member-user", ProjectID: "project-1", UserID: "user-1", ProjectRole: app.ProjectRoleViewer}).Error)
}

func aiProviderClaimsMiddleware(userID, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &app.Claims{UserID: userID, Role: role})
		c.Next()
	}
}

func newAIProviderHandlerRouter(userID, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(aiProviderClaimsMiddleware(userID, role))

	authorized := r.Group("/api/v1")
	authorized.Use(func(c *gin.Context) {
		c.Next()
	})
	users := authorized.Group("/users")
	users.GET("/me/ai-providers", ListCurrentUserAIProviders)
	users.POST("/me/ai-providers", CreateCurrentUserAIProvider)
	users.PUT("/me/ai-providers/:providerID", UpdateCurrentUserAIProvider)
	users.DELETE("/me/ai-providers/:providerID", DeleteCurrentUserAIProvider)

	projects := authorized.Group("/projects")
	projectsRead := projects.Group("", middlewares.RequireProjectRole(app.ProjectRoleViewer))
	projectsRead.GET("/:projectID/ai-providers", ListProjectAIProviders)
	projectsRead.GET("/:projectID/builder-model-options", ListBuilderAvailableModelOptions)
	projectsWrite := projects.Group("", middlewares.RequireProjectRole(app.ProjectRoleOwner))
	projectsWrite.POST("/:projectID/ai-providers", CreateProjectAIProvider)
	projectsWrite.PUT("/:projectID/ai-providers/:providerID", UpdateProjectAIProvider)
	projectsWrite.DELETE("/:projectID/ai-providers/:providerID", DeleteProjectAIProvider)

	return r
}

type aiProviderAPIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func TestCurrentUserAIProviderHandlers(t *testing.T) {
	setupAIProviderHandlerTestDB(t)
	originalConfig := app.Config
	app.Config = app.AppConfig{
		BuilderProviderRegistryJSON: `[
			{"key":"anthropic-project","base_url":"https://api.anthropic.com","api_key":"shared-secret"},
			{"key":"openai-user","base_url":"https://api.openai.com","api_key":"secret-key"}
		]`,
		BuilderModelProfileRegistryJSON: `[
			{"key":"claude-sonnet-4","model":"claude-4-sonnet"},
			{"key":"gpt-4.1","model":"gpt-4.1"}
		]`,
		BuilderDefaultProviderKey:     "anthropic-project",
		BuilderDefaultModelProfileKey: "claude-sonnet-4",
	}
	t.Cleanup(func() { app.Config = originalConfig })

	r := newAIProviderHandlerRouter("user-1", app.UserRoleUser)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/ai-providers", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)

	createReqBody := map[string]any{
		"provider_key":              "openai-user",
		"display_name":              "OpenAI Personal",
		"base_url":                  "https://api.openai.com",
		"api_key":                   "secret-key",
		"default_model_profile_key": "gpt-4.1",
		"enabled":                   true,
		"is_default":                true,
	}
	createBody, err := json.Marshal(createReqBody)
	require.NoError(t, err)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/ai-providers", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var createResp aiProviderAPIResponse
	require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &createResp))
	assert.Empty(t, createResp.Error)

	var created models.AIProviderResponse
	require.NoError(t, json.Unmarshal(createResp.Data, &created))
	assert.True(t, created.IsDefault)

	updateReqBody := map[string]any{
		"provider_key":              "openai-user",
		"display_name":              "OpenAI Personal Updated",
		"base_url":                  "https://api.openai.com",
		"api_key":                   "secret-key-2",
		"default_model_profile_key": "gpt-4.1",
		"enabled":                   true,
		"is_default":                false,
	}
	updateBody, err := json.Marshal(updateReqBody)
	require.NoError(t, err)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/ai-providers/"+created.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)
	require.Equal(t, http.StatusOK, updateW.Code)
	var updateResp aiProviderAPIResponse
	require.NoError(t, json.Unmarshal(updateW.Body.Bytes(), &updateResp))
	var updated models.AIProviderResponse
	require.NoError(t, json.Unmarshal(updateResp.Data, &updated))
	assert.False(t, updated.IsDefault)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/ai-providers/"+created.ID, nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	require.Equal(t, http.StatusNoContent, deleteW.Code)
}

func TestCurrentUserAIProviderHandlersAllowDirectBuilderAliases(t *testing.T) {
	setupAIProviderHandlerTestDB(t)
	r := newAIProviderHandlerRouter("user-1", app.UserRoleUser)

	t.Run("accepts arbitrary provider alias on create", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"provider_key":              "missing-provider",
			"display_name":              "Broken Provider",
			"base_url":                  "https://api.example.com",
			"api_key":                   "secret-key",
			"default_model_profile_key": "gpt-4.1",
			"enabled":                   true,
			"is_default":                true,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/ai-providers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "missing-provider")
	})

	t.Run("accepts arbitrary model key on update", func(t *testing.T) {
		provider := entities.UserAIProvider{
			ID:                     "user-provider-1",
			UserID:                 "user-1",
			ProviderKey:            "openai-user",
			DisplayName:            "OpenAI Personal",
			BaseURL:                "https://api.openai.com",
			APIKey:                 "secret-key",
			DefaultModelProfileKey: "gpt-4.1",
			Enabled:                true,
			IsDefault:              true,
		}
		require.NoError(t, db.DB.Create(&provider).Error)

		body, err := json.Marshal(map[string]any{
			"provider_key":              "openai-user",
			"display_name":              "OpenAI Personal",
			"base_url":                  "https://api.openai.com",
			"api_key":                   "secret-key",
			"default_model_profile_key": "missing-model",
			"enabled":                   true,
			"is_default":                true,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/ai-providers/user-provider-1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "missing-model")
	})
}

func TestProjectAIProviderHandlers(t *testing.T) {
	setupAIProviderHandlerTestDB(t)
	originalConfig := app.Config
	app.Config = app.AppConfig{
		BuilderProviderRegistryJSON: `[
			{"key":"anthropic-project","base_url":"https://api.anthropic.com","api_key":"shared-secret"},
			{"key":"openai-user","base_url":"https://api.openai.com","api_key":"secret-key"}
		]`,
		BuilderModelProfileRegistryJSON: `[
			{"key":"claude-sonnet-4","model":"claude-4-sonnet"},
			{"key":"gpt-4.1","model":"gpt-4.1"}
		]`,
		BuilderDefaultProviderKey:     "anthropic-project",
		BuilderDefaultModelProfileKey: "claude-sonnet-4",
	}
	t.Cleanup(func() { app.Config = originalConfig })

	ownerRouter := newAIProviderHandlerRouter("owner-1", app.UserRoleUser)
	viewerRouter := newAIProviderHandlerRouter("user-1", app.UserRoleUser)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/ai-providers", nil)
	listW := httptest.NewRecorder()
	viewerRouter.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)

	createReqBody := map[string]any{
		"provider_key":              "anthropic-project",
		"display_name":              "Anthropic Shared",
		"base_url":                  "https://api.anthropic.com",
		"api_key":                   "shared-secret",
		"default_model_profile_key": "claude-sonnet-4",
		"enabled":                   true,
		"is_default":                true,
	}
	createBody, err := json.Marshal(createReqBody)
	require.NoError(t, err)

	ownerReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/ai-providers", bytes.NewReader(createBody))
	ownerReq.Header.Set("Content-Type", "application/json")
	ownerW := httptest.NewRecorder()
	ownerRouter.ServeHTTP(ownerW, ownerReq)
	require.Equal(t, http.StatusCreated, ownerW.Code)

	viewerReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/ai-providers", bytes.NewReader(createBody))
	viewerReq.Header.Set("Content-Type", "application/json")
	viewerW := httptest.NewRecorder()
	viewerRouter.ServeHTTP(viewerW, viewerReq)
	require.Equal(t, http.StatusForbidden, viewerW.Code)

	var ownerResp aiProviderAPIResponse
	require.NoError(t, json.Unmarshal(ownerW.Body.Bytes(), &ownerResp))
	assert.Empty(t, ownerResp.Error)

	var created models.AIProviderResponse
	require.NoError(t, json.Unmarshal(ownerResp.Data, &created))
	assert.True(t, created.IsDefault)

	updateReqBody := map[string]any{
		"provider_key":              "anthropic-project",
		"display_name":              "Anthropic Shared Updated",
		"base_url":                  "https://api.anthropic.com",
		"api_key":                   "shared-secret-2",
		"default_model_profile_key": "claude-sonnet-4",
		"enabled":                   true,
		"is_default":                false,
	}
	updateBody, err := json.Marshal(updateReqBody)
	require.NoError(t, err)
	ownerUpdateReq := httptest.NewRequest(http.MethodPut, "/api/v1/projects/project-1/ai-providers/"+created.ID, bytes.NewReader(updateBody))
	ownerUpdateReq.Header.Set("Content-Type", "application/json")
	ownerUpdateW := httptest.NewRecorder()
	ownerRouter.ServeHTTP(ownerUpdateW, ownerUpdateReq)
	require.Equal(t, http.StatusOK, ownerUpdateW.Code)
	var ownerUpdateResp aiProviderAPIResponse
	require.NoError(t, json.Unmarshal(ownerUpdateW.Body.Bytes(), &ownerUpdateResp))
	var updated models.AIProviderResponse
	require.NoError(t, json.Unmarshal(ownerUpdateResp.Data, &updated))
	assert.False(t, updated.IsDefault)

	viewerUpdateReq := httptest.NewRequest(http.MethodPut, "/api/v1/projects/project-1/ai-providers/"+created.ID, bytes.NewReader(updateBody))
	viewerUpdateReq.Header.Set("Content-Type", "application/json")
	viewerUpdateW := httptest.NewRecorder()
	viewerRouter.ServeHTTP(viewerUpdateW, viewerUpdateReq)
	require.Equal(t, http.StatusForbidden, viewerUpdateW.Code)

	ownerDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/project-1/ai-providers/"+created.ID, nil)
	ownerDeleteW := httptest.NewRecorder()
	ownerRouter.ServeHTTP(ownerDeleteW, ownerDeleteReq)
	require.Equal(t, http.StatusNoContent, ownerDeleteW.Code)

	viewerDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/project-1/ai-providers/"+created.ID, nil)
	viewerDeleteW := httptest.NewRecorder()
	viewerRouter.ServeHTTP(viewerDeleteW, viewerDeleteReq)
	require.Equal(t, http.StatusForbidden, viewerDeleteW.Code)
}

func TestProjectAIProviderHandlersAllowDirectBuilderAliases(t *testing.T) {
	setupAIProviderHandlerTestDB(t)
	ownerRouter := newAIProviderHandlerRouter("owner-1", app.UserRoleUser)

	t.Run("accepts arbitrary provider alias on create", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"provider_key":              "missing-provider",
			"display_name":              "Broken Provider",
			"base_url":                  "https://api.example.com",
			"api_key":                   "shared-secret",
			"default_model_profile_key": "claude-sonnet-4",
			"enabled":                   true,
			"is_default":                true,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/ai-providers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ownerRouter.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "missing-provider")
	})

	t.Run("accepts arbitrary model key on update", func(t *testing.T) {
		provider := entities.ProjectAIProvider{
			ID:                     "project-provider-1",
			ProjectID:              "project-1",
			ProviderKey:            "anthropic-project",
			DisplayName:            "Anthropic Shared",
			BaseURL:                "https://api.anthropic.com",
			APIKey:                 "shared-secret",
			DefaultModelProfileKey: "claude-sonnet-4",
			Enabled:                true,
			IsDefault:              true,
		}
		require.NoError(t, db.DB.Create(&provider).Error)

		body, err := json.Marshal(map[string]any{
			"provider_key":              "anthropic-project",
			"display_name":              "Anthropic Shared",
			"base_url":                  "https://api.anthropic.com",
			"api_key":                   "shared-secret",
			"default_model_profile_key": "missing-model",
			"enabled":                   true,
			"is_default":                true,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/project-1/ai-providers/project-provider-1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ownerRouter.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "missing-model")
	})
}

func TestBuilderAvailableModelsHandlerReturnsEnabledDatabaseProviders(t *testing.T) {
	setupAIProviderHandlerTestDB(t)
	require.NoError(t, db.DB.Create(&entities.ProjectAIProvider{
		ID:                     "project-provider-invalid",
		ProjectID:              "project-1",
		ProviderKey:            "missing-provider",
		DisplayName:            "Broken Project Provider",
		BaseURL:                "https://broken.example.com",
		APIKey:                 "broken-secret",
		DefaultModelProfileKey: "missing-model",
		Enabled:                true,
	}).Error)

	r := newAIProviderHandlerRouter("user-1", app.UserRoleUser)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-model-options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp aiProviderAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)
	assert.Contains(t, string(resp.Data), "missing-provider")
	assert.Contains(t, string(resp.Data), "missing-model")
}

func TestBuilderAvailableModelsHandler(t *testing.T) {
	setupAIProviderHandlerTestDB(t)
	originalConfig := app.Config
	app.Config = app.AppConfig{
		BuilderProviderRegistryJSON: `[
			{"key":"anthropic-project","base_url":"https://api.anthropic.com","api_key":"shared-secret"},
			{"key":"openai-user","base_url":"https://api.openai.com","api_key":"secret-key"}
		]`,
		BuilderModelProfileRegistryJSON: `[
			{"key":"claude-sonnet-4","model":"claude-4-sonnet"},
			{"key":"gpt-4.1","model":"gpt-4.1"}
		]`,
		BuilderDefaultProviderKey:     "anthropic-project",
		BuilderDefaultModelProfileKey: "claude-sonnet-4",
	}
	t.Cleanup(func() { app.Config = originalConfig })
	require.NoError(t, db.DB.Create(&entities.ProjectAIProvider{
		ID:                     "project-provider-1",
		ProjectID:              "project-1",
		ProviderKey:            "anthropic-project",
		DisplayName:            "Anthropic Shared",
		BaseURL:                "https://api.anthropic.com",
		APIKey:                 "shared-secret",
		DefaultModelProfileKey: "claude-sonnet-4",
		Enabled:                true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.UserAIProvider{
		ID:                     "user-provider-1",
		UserID:                 "user-1",
		ProviderKey:            "openai-user",
		DisplayName:            "OpenAI Personal",
		BaseURL:                "https://api.openai.com",
		APIKey:                 "secret-key",
		DefaultModelProfileKey: "gpt-4.1",
		Enabled:                true,
	}).Error)

	r := newAIProviderHandlerRouter("user-1", app.UserRoleUser)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-model-options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
