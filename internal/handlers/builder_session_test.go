package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type builderSessionAPIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

type builderSessionDownloadRecorder struct {
	HeaderMap http.Header
	Status    int
	Body      strings.Builder
}

func (r *builderSessionDownloadRecorder) Header() http.Header {
	if r.HeaderMap == nil {
		r.HeaderMap = make(http.Header)
	}
	return r.HeaderMap
}

func (r *builderSessionDownloadRecorder) Write(data []byte) (int, error) {
	return r.Body.Write(data)
}

func (r *builderSessionDownloadRecorder) WriteHeader(statusCode int) {
	r.Status = statusCode
}

type builderStreamHookRecorder struct {
	*httptest.ResponseRecorder
	flushHook func()
}

func (r *builderStreamHookRecorder) Flush() {
	if r.flushHook != nil {
		hook := r.flushHook
		r.flushHook = nil
		hook()
	}
	r.ResponseRecorder.Flush()
}

func setupBuilderSessionHandlerTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	originalConfig := app.Config
	t.Cleanup(func() {
		db.DB = originalDB
		app.Config = originalConfig
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	sqlDB, err := testDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	db.DB = testDB
	require.NoError(t, db.Migrate())
}

func builderSessionClaimsMiddleware(userID, username, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID:   userID,
			Username: username,
			Role:     role,
		})
		c.Next()
	}
}

func newBuilderSessionHandlerRouter(userID, username, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(builderSessionClaimsMiddleware(userID, username, role))

	projectsRead := r.Group("/api/v1/projects", middlewares.RequireProjectRole(app.ProjectRoleViewer))
	projectsRead.GET("/:projectID/builder-model-options", ListBuilderAvailableModelOptions)
	projectsRead.GET("/:projectID/builder-model-selection", GetBuilderDefaultModelSelection)
	projectsRead.GET("/:projectID/builder-sessions", ListBuilderSessions)
	projectsRead.GET("/:projectID/builder-sessions/:sessionID", GetBuilderSession)
	projectsRead.GET("/:projectID/builder-sessions/:sessionID/preview", GetBuilderSessionPreview)
	projectsRead.GET("/:projectID/builder-sessions/:sessionID/runs/:runID/preview/launch", LaunchBuilderSessionPreview)
	projectsRead.GET("/:projectID/builder-sessions/:sessionID/runs/:runID/delivery/download", DownloadBuilderSessionSnapshot)
	r.GET("/builder-preview/projects/:projectID/sessions/:sessionID/runs/:runID/*assetPath", middlewares.BuilderPreviewAuth(), middlewares.RequireProjectRole(app.ProjectRoleViewer), ReadBuilderPreviewAsset)

	projectsWrite := r.Group("/api/v1/projects", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
	projectsWrite.POST("/:projectID/builder-sessions", CreateBuilderSession)
	projectsWrite.POST("/:projectID/builder-sessions/:sessionID/messages", PostBuilderSessionMessage)
	projectsWrite.POST("/:projectID/builder-sessions/:sessionID/runs/:runID/cancel", RequestBuilderRunCancel)
	projectsWrite.GET("/:projectID/builder-sessions/:sessionID/runs/:runID/logs", StreamBuilderRunLogs)

	return r
}

func newBuilderSessionDirectWriteRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/v1/projects/:projectID/builder-sessions", CreateBuilderSession)
	r.POST("/api/v1/projects/:projectID/builder-sessions/:sessionID/messages", PostBuilderSessionMessage)
	r.POST("/api/v1/projects/:projectID/builder-sessions/:sessionID/runs/:runID/cancel", RequestBuilderRunCancel)

	return r
}

func seedBuilderSessionProjectMember(t *testing.T, projectID, userID, projectRole string) {
	t.Helper()

	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          projectID + "-" + userID,
		ProjectID:   projectID,
		UserID:      userID,
		ProjectRole: projectRole,
	}).Error)
}

func seedBuilderSessionListFixture(t *testing.T) {
	t.Helper()

	now := time.Now().UTC()
	workspaceID := "workspace-1"

	sessions := []entities.BuilderSession{
		{
			Base: entities.Base{
				ID:        "session-1",
				CreatedAt: now.Add(-8 * time.Minute),
				UpdatedAt: now.Add(-7 * time.Minute),
			},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Older builder session",
			Summary:        "Older summary",
			Status:         entities.BuilderSessionStatusRunning,
			CreatedBy:      "user-1",
			LastActivityAt: now.Add(-6 * time.Minute),
		},
		{
			Base: entities.Base{
				ID:        "session-2",
				CreatedAt: now.Add(-2 * time.Minute),
				UpdatedAt: now.Add(-2 * time.Minute),
			},
			ProjectID:      "project-1",
			BuildEnvID:     "env-2",
			Title:          "Newer builder session",
			Summary:        "Newer summary",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-2",
			LastActivityAt: now.Add(-time.Minute),
		},
		{
			Base: entities.Base{
				ID:        "session-3",
				CreatedAt: now.Add(-time.Minute),
				UpdatedAt: now.Add(-time.Minute),
			},
			ProjectID:      "project-2",
			BuildEnvID:     "env-3",
			Title:          "Other project session",
			Summary:        "Other project summary",
			Status:         entities.BuilderSessionStatusArchived,
			CreatedBy:      "user-3",
			LastActivityAt: now,
		},
	}
	require.NoError(t, db.DB.Create(&sessions).Error)

	runs := []entities.BuilderRun{
		{
			ID:                 "run-1",
			CreatedAt:          now.Add(-7 * time.Minute),
			UpdatedAt:          now.Add(-7 * time.Minute),
			SessionID:          "session-1",
			TriggerMessageID:   "message-1",
			WorkspaceID:        &workspaceID,
			Status:             entities.BuilderRunStatusExecuting,
			RequestedBy:        "user-1",
			InstructionSummary: "Older summary",
		},
		{
			ID:                 "run-2",
			CreatedAt:          now.Add(-2 * time.Minute),
			UpdatedAt:          now.Add(-2 * time.Minute),
			SessionID:          "session-2",
			TriggerMessageID:   "message-2",
			Status:             entities.BuilderRunStatusQueued,
			RequestedBy:        "user-2",
			InstructionSummary: "Newer summary",
		},
	}
	require.NoError(t, db.DB.Create(&runs).Error)

	require.NoError(t, db.DB.Create(&entities.BuilderWorkspace{
		ID:            workspaceID,
		CreatedAt:     now.Add(-5 * time.Minute),
		UpdatedAt:     now.Add(-5 * time.Minute),
		SessionID:     "session-1",
		BuildEnvID:    "env-1",
		ClusterID:     "cluster-1",
		Namespace:     "builder-session-1",
		PodName:       "builder-session-1-0",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: "/workspace/session-1",
	}).Error)

	require.NoError(t, db.DB.Create(&entities.BuilderArtifact{
		ID:           "artifact-1",
		CreatedAt:    now.Add(-4 * time.Minute),
		UpdatedAt:    now.Add(-4 * time.Minute),
		SessionID:    "session-1",
		WorkspaceID:  workspaceID,
		RunID:        "run-1",
		Kind:         "file",
		Path:         "plans/older-plan.md",
		MetadataJSON: `{"size_bytes":120}`,
	}).Error)
}

func seedBuilderSessionDetailFixture(t *testing.T) string {
	t.Helper()

	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	terminatedAt := now.Add(-2 * time.Minute)

	session := entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-detail-1",
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Summary:        "Builder session summary",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
		ExpiresAt:      &expiresAt,
	}
	require.NoError(t, db.DB.Create(&session).Error)

	messageOne := entities.BuilderMessage{
		ID:        "message-1",
		CreatedAt: now.Add(-9 * time.Minute),
		UpdatedAt: now.Add(-9 * time.Minute),
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Create the API layout.",
		CreatedBy: "user-1",
	}
	messageTwo := entities.BuilderMessage{
		ID:        "message-2",
		CreatedAt: now.Add(-8 * time.Minute),
		UpdatedAt: now.Add(-8 * time.Minute),
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Add query models.",
		CreatedBy: "user-1",
	}
	require.NoError(t, db.DB.Create(&messageOne).Error)
	require.NoError(t, db.DB.Create(&messageTwo).Error)

	workspaceOneID := "workspace-1"
	workspaceTwoID := "workspace-2"
	runOne := entities.BuilderRun{
		ID:                 "run-1",
		CreatedAt:          now.Add(-7 * time.Minute),
		UpdatedAt:          now.Add(-7 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   messageOne.ID,
		WorkspaceID:        &workspaceOneID,
		Status:             entities.BuilderRunStatusExecuting,
		RequestedBy:        "user-1",
		InstructionSummary: "Create the API layout.",
		ExecutionLog:       "Cloning repository",
	}
	runTwo := entities.BuilderRun{
		ID:                 "run-2",
		CreatedAt:          now.Add(-6 * time.Minute),
		UpdatedAt:          now.Add(-6 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   messageTwo.ID,
		WorkspaceID:        &workspaceTwoID,
		Status:             entities.BuilderRunStatusSucceeded,
		RequestedBy:        "user-1",
		InstructionSummary: "Add query models.",
		ExecutionLog:       "Applied builder diff",
	}
	require.NoError(t, db.DB.Create(&runOne).Error)
	require.NoError(t, db.DB.Create(&runTwo).Error)

	assistantRunID := runTwo.ID
	require.NoError(t, db.DB.Create(&entities.BuilderMessage{
		ID:           "message-3",
		CreatedAt:    now.Add(-5 * time.Minute),
		UpdatedAt:    now.Add(-5 * time.Minute),
		SessionID:    session.ID,
		RunID:        &assistantRunID,
		Role:         entities.BuilderMessageRoleAssistant,
		Content:      "Query models added.",
		MetadataJSON: `{"source":"builder-agent"}`,
		CreatedBy:    "user-1",
	}).Error)

	require.NoError(t, db.DB.Create(&entities.BuilderWorkspace{
		ID:            workspaceOneID,
		CreatedAt:     now.Add(-3 * time.Minute),
		UpdatedAt:     now.Add(-3 * time.Minute),
		SessionID:     session.ID,
		BuildEnvID:    "env-1",
		ClusterID:     "cluster-1",
		Namespace:     "builder-session-1-v1",
		PodName:       "builder-session-1-v1-0",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusExpired,
		WorkspaceRoot: "/workspace/session-1-v1",
		TerminatedAt:  &terminatedAt,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderWorkspace{
		ID:            workspaceTwoID,
		CreatedAt:     now.Add(-4 * time.Minute),
		UpdatedAt:     now.Add(-4 * time.Minute),
		SessionID:     session.ID,
		BuildEnvID:    "env-1",
		ClusterID:     "cluster-1",
		Namespace:     "builder-session-1-v2",
		PodName:       "builder-session-1-v2-0",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: "/workspace/session-1-v2",
	}).Error)

	require.NoError(t, db.DB.Create(&entities.BuilderArtifact{
		ID:           "artifact-1",
		CreatedAt:    now.Add(-3 * time.Minute),
		UpdatedAt:    now.Add(-3 * time.Minute),
		SessionID:    session.ID,
		WorkspaceID:  workspaceOneID,
		RunID:        runOne.ID,
		Kind:         "file",
		Path:         "plans/old-plan.md",
		MetadataJSON: `{"size_bytes":120}`,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderArtifact{
		ID:           "artifact-2",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now.Add(-2 * time.Minute),
		SessionID:    session.ID,
		WorkspaceID:  workspaceTwoID,
		RunID:        runTwo.ID,
		Kind:         "file",
		Path:         "plans/new-plan.md",
		MetadataJSON: `{"size_bytes":256}`,
	}).Error)

	return session.ID
}

func seedBuilderSessionTerminatedWorkspaceArtifactFixture(t *testing.T) string {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	terminatedAt := now.Add(-30 * time.Second)
	workspaceID := "workspace-terminated-handler"
	sessionID := "session-terminated-handler"
	runID := "run-terminated-handler"

	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base: entities.Base{
			ID:        sessionID,
			CreatedAt: now.Add(-5 * time.Minute),
			UpdatedAt: now.Add(-4 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Completed build session",
		Summary:        "Build outputs should remain visible after workspace teardown.",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}).Error)

	require.NoError(t, db.DB.Create(&entities.BuilderMessage{
		ID:        "message-terminated-handler",
		CreatedAt: now.Add(-4 * time.Minute),
		UpdatedAt: now.Add(-4 * time.Minute),
		SessionID: sessionID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Build the generated app.",
		CreatedBy: "user-1",
	}).Error)

	completedAt := now.Add(-2 * time.Minute)
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 runID,
		CreatedAt:          now.Add(-3 * time.Minute),
		UpdatedAt:          now.Add(-2 * time.Minute),
		SessionID:          sessionID,
		TriggerMessageID:   "message-terminated-handler",
		WorkspaceID:        &workspaceID,
		Status:             entities.BuilderRunStatusSucceeded,
		RequestedBy:        "user-1",
		InstructionSummary: "Build the generated app.",
		CompletedAt:        &completedAt,
	}).Error)

	require.NoError(t, db.DB.Create(&entities.BuilderWorkspace{
		ID:            workspaceID,
		CreatedAt:     now.Add(-3 * time.Minute),
		UpdatedAt:     now.Add(-2 * time.Minute),
		SessionID:     sessionID,
		BuildEnvID:    "env-1",
		ClusterID:     "cluster-1",
		Namespace:     "builder-session-terminated-handler",
		PodName:       "builder-session-terminated-handler-0",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusExpired,
		WorkspaceRoot: "/workspace/session-terminated-handler",
		TerminatedAt:  &terminatedAt,
	}).Error)

	require.NoError(t, db.DB.Create(&[]entities.BuilderArtifact{
		{
			ID:           "artifact-terminated-handler-source",
			CreatedAt:    now.Add(-2 * time.Minute),
			UpdatedAt:    now.Add(-2 * time.Minute),
			SessionID:    sessionID,
			WorkspaceID:  workspaceID,
			RunID:        runID,
			Kind:         entities.BuilderArtifactKindWorkspaceFile,
			Path:         "src/main.tsx",
			MetadataJSON: `{"size_bytes":120,"source_phase":"materializing_files"}`,
		},
		{
			ID:           "artifact-terminated-handler-build",
			CreatedAt:    now.Add(-time.Minute),
			UpdatedAt:    now.Add(-time.Minute),
			SessionID:    sessionID,
			WorkspaceID:  workspaceID,
			RunID:        runID,
			Kind:         entities.BuilderArtifactKindBuildOutput,
			Path:         "dist/index.html",
			MetadataJSON: `{"size_bytes":512,"output_root":"dist","source_phase":"building"}`,
		},
	}).Error)

	return sessionID
}

func TestCreateBuilderSessionHandler(t *testing.T) {
	t.Run("fails fast when claims are missing", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)

		r := newBuilderSessionDirectWriteRouter()
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/projects/project-1/builder-sessions",
			strings.NewReader(`{"build_env_id":"env-1","title":"Bootstrap API","prompt":"Create the first builder workflow."}`),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "unauthorized", resp.Error)

		var sessionCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderSession{}).Count(&sessionCount).Error)
		assert.Equal(t, int64(0), sessionCount)
	})

	t.Run("maps stable builder service errors through the shared status helper", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, builderSessionErrorStatus(&services.BuilderSessionNotFoundError{ProjectID: "project-1", SessionID: "session-1"}))
		assert.Equal(t, http.StatusConflict, builderSessionErrorStatus(&services.BuilderSessionNotAppendableError{SessionID: "session-1", Status: entities.BuilderSessionStatusArchived}))
	})

	t.Run("creates a builder session for project developers", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)

		r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/projects/project-1/builder-sessions",
			strings.NewReader(`{"build_env_id":"env-1","title":"Bootstrap API","prompt":"Create the first builder workflow."}`),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.Error)

		var detail models.BuilderSessionDetailResponse
		require.NoError(t, json.Unmarshal(resp.Data, &detail))
		assert.Equal(t, "project-1", detail.Session.ProjectID)
		assert.Equal(t, "env-1", detail.Session.BuildEnvID)
		assert.Equal(t, "Bootstrap API", detail.Session.Title)
		assert.Equal(t, string(entities.BuilderSessionStatusProvisioning), detail.Session.Status)
		assert.Equal(t, string(entities.BuilderRunStatusQueued), detail.Session.LatestRunStatus)
		require.Len(t, detail.Messages, 1)
		assert.Equal(t, "Create the first builder workflow.", detail.Messages[0].Content)
		require.Len(t, detail.Runs, 1)
		assert.Equal(t, string(entities.BuilderRunStatusQueued), detail.Runs[0].Status)
		assert.Nil(t, detail.Workspace)
		assert.Empty(t, detail.Artifacts)
	})

	t.Run("rejects viewers before the handler runs", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/projects/project-1/builder-sessions",
			strings.NewReader(`{"build_env_id":"env-1","prompt":"Create the first builder workflow."}`),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "insufficient permissions", resp.Error)
	})
}

func TestListBuilderAvailableModelOptionsHandler(t *testing.T) {
	setupBuilderSessionHandlerTestDB(t)
	seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
	originalConfig := app.Config
	app.Config = app.AppConfig{
		BuilderProviderRegistryJSON: `[{
			"key":"anthropic-project","base_url":"https://api.anthropic.com","api_key":"shared-secret"},
			{"key":"openai-user","base_url":"https://api.openai.com","api_key":"secret-key"}
		]`,
		BuilderModelProfileRegistryJSON: `[{
			"key":"claude-sonnet-4","model":"claude-4-sonnet"},
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
		UserID:                 "viewer-1",
		ProviderKey:            "openai-user",
		DisplayName:            "OpenAI Personal",
		BaseURL:                "https://api.openai.com",
		APIKey:                 "secret-key",
		DefaultModelProfileKey: "gpt-4.1",
		Enabled:                true,
	}).Error)

	r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-model-options", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp builderSessionAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Empty(t, resp.Error)
	assert.Contains(t, string(resp.Data), "anthropic-project")
	assert.Contains(t, string(resp.Data), "openai-user")
}

func TestGetBuilderDefaultModelSelection(t *testing.T) {
	setupBuilderSessionHandlerTestDB(t)
	seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
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
		ID:                     "project-provider-default",
		ProjectID:              "project-1",
		ProviderKey:            "anthropic-project",
		DisplayName:            "Anthropic Shared",
		BaseURL:                "https://api.anthropic.com",
		APIKey:                 "shared-secret",
		DefaultModelProfileKey: "claude-sonnet-4",
		Enabled:                true,
		IsDefault:              true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.UserAIProvider{
		ID:                     "user-provider-default",
		UserID:                 "viewer-1",
		ProviderKey:            "openai-user",
		DisplayName:            "OpenAI Personal",
		BaseURL:                "https://api.openai.com",
		APIKey:                 "secret-key",
		DefaultModelProfileKey: "gpt-4.1",
		Enabled:                true,
		IsDefault:              true,
	}).Error)

	r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-model-selection", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp builderSessionAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Empty(t, resp.Error)
	assert.Contains(t, string(resp.Data), "effective_default_source")
	assert.Contains(t, string(resp.Data), "project")
	assert.Contains(t, string(resp.Data), "anthropic-project")
}

func TestPostBuilderSessionMessageHandler(t *testing.T) {
	t.Run("fails fast when claims are missing", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-1"},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Bootstrap API",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: time.Now().UTC().Add(-time.Minute),
		}).Error)

		r := newBuilderSessionDirectWriteRouter()
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/projects/project-1/builder-sessions/session-1/messages",
			strings.NewReader(`{"content":"Add pagination to the service list."}`),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "unauthorized", resp.Error)

		var messageCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderMessage{}).Where("session_id = ?", "session-1").Count(&messageCount).Error)
		assert.Equal(t, int64(0), messageCount)

		var runCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("session_id = ?", "session-1").Count(&runCount).Error)
		assert.Equal(t, int64(0), runCount)
	})

	t.Run("appends a builder message for project developers", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)

		lastActivityAt := time.Now().UTC().Add(-10 * time.Minute)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-1"},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Bootstrap API",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: lastActivityAt,
		}).Error)

		r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/projects/project-1/builder-sessions/session-1/messages",
			strings.NewReader(`{"content":"Add pagination to the service list."}`),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.Error)

		var detail models.BuilderSessionDetailResponse
		require.NoError(t, json.Unmarshal(resp.Data, &detail))
		assert.Equal(t, "session-1", detail.Session.ID)
		assert.Equal(t, string(entities.BuilderSessionStatusReady), detail.Session.Status)
		assert.Equal(t, string(entities.BuilderRunStatusQueued), detail.Session.LatestRunStatus)
		require.Len(t, detail.Messages, 1)
		assert.Equal(t, "Add pagination to the service list.", detail.Messages[0].Content)
		require.Len(t, detail.Runs, 1)
		assert.Equal(t, string(entities.BuilderRunStatusQueued), detail.Runs[0].Status)
		assert.Nil(t, detail.Runs[0].StartedAt)
	})

	t.Run("maps missing sessions to not found", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)

		r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/projects/project-1/builder-sessions/missing-session/messages",
			strings.NewReader(`{"content":"Should fail."}`),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Error, "builder session missing-session not found")
	})

	t.Run("maps closed sessions to conflict", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-archived"},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Archived session",
			Status:         entities.BuilderSessionStatusArchived,
			CreatedBy:      "user-1",
			LastActivityAt: time.Now().UTC().Add(-time.Minute),
		}).Error)

		r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/projects/project-1/builder-sessions/session-archived/messages",
			strings.NewReader(`{"content":"Should be rejected."}`),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusConflict, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Error, "is not appendable")
	})

	t.Run("rejects viewers before appending messages", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-1"},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Bootstrap API",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: time.Now().UTC().Add(-time.Minute),
		}).Error)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/projects/project-1/builder-sessions/session-1/messages",
			strings.NewReader(`{"content":"Should be forbidden."}`),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "insufficient permissions", resp.Error)
	})
}

func TestListBuilderSessionsHandler(t *testing.T) {
	t.Run("lists project-scoped builder sessions for viewers", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		seedBuilderSessionListFixture(t)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.Error)

		var items []models.BuilderSessionListItem
		require.NoError(t, json.Unmarshal(resp.Data, &items))
		require.Len(t, items, 2)
		assert.Equal(t, "session-2", items[0].ID)
		assert.Equal(t, "Newer builder session", items[0].Title)
		assert.Equal(t, string(entities.BuilderRunStatusQueued), items[0].LatestRunStatus)
		assert.Equal(t, int64(0), items[0].ArtifactCount)
		assert.Equal(t, "session-1", items[1].ID)
		assert.Equal(t, string(entities.BuilderWorkspaceStatusActive), items[1].CurrentWorkspaceStatus)
		assert.Equal(t, "/workspace/session-1", items[1].CurrentWorkspaceRoot)
		assert.Equal(t, int64(1), items[1].ArtifactCount)
	})
}

func TestRequestBuilderRunCancellationHandler(t *testing.T) {
	t.Run("persists cancel intent without changing the session lifecycle", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)

		now := time.Now().UTC().Truncate(time.Second)
		queuedPhase := entities.BuilderRunPhaseQueued
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base: entities.Base{
				ID:        "session-1",
				CreatedAt: now.Add(-2 * time.Minute),
				UpdatedAt: now.Add(-2 * time.Minute),
			},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Cancelable session",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: now.Add(-time.Minute),
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderMessage{
			ID:        "message-1",
			CreatedAt: now.Add(-90 * time.Second),
			UpdatedAt: now.Add(-90 * time.Second),
			SessionID: "session-1",
			Role:      entities.BuilderMessageRoleUser,
			Content:   "Cancel prompt",
			CreatedBy: "user-1",
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderRun{
			ID:                 "run-1",
			CreatedAt:          now.Add(-time.Minute),
			UpdatedAt:          now.Add(-time.Minute),
			SessionID:          "session-1",
			TriggerMessageID:   "message-1",
			Status:             entities.BuilderRunStatusQueued,
			Phase:              &queuedPhase,
			RequestedBy:        "user-1",
			InstructionSummary: "Cancel prompt",
		}).Error)

		r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/builder-sessions/session-1/runs/run-1/cancel", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.Error)

		var detail models.BuilderSessionDetailResponse
		require.NoError(t, json.Unmarshal(resp.Data, &detail))
		assert.Equal(t, "session-1", detail.Session.ID)
		assert.Equal(t, string(entities.BuilderSessionStatusReady), detail.Session.Status)
		require.Len(t, detail.Runs, 1)
		assert.Equal(t, "run-1", detail.Runs[0].ID)
		assert.Equal(t, string(entities.BuilderRunStatusQueued), detail.Runs[0].Status)

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", "run-1").Error)
		require.NotNil(t, persistedRun.CancelRequestedAt)
		assert.Equal(t, entities.BuilderRunStatusQueued, persistedRun.Status)

		var session entities.BuilderSession
		require.NoError(t, db.DB.First(&session, "id = ?", "session-1").Error)
		assert.Equal(t, entities.BuilderSessionStatusReady, session.Status)
	})

	t.Run("maps out-of-scope runs to not found", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)

		now := time.Now().UTC().Truncate(time.Second)
		queuedPhase := entities.BuilderRunPhaseQueued
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-1", CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute)},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Primary session",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: now.Add(-time.Minute),
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-2", CreatedAt: now.Add(-90 * time.Second), UpdatedAt: now.Add(-90 * time.Second)},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Secondary session",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: now.Add(-30 * time.Second),
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderMessage{
			ID:        "message-2",
			CreatedAt: now.Add(-60 * time.Second),
			UpdatedAt: now.Add(-60 * time.Second),
			SessionID: "session-2",
			Role:      entities.BuilderMessageRoleUser,
			Content:   "Scoped cancel prompt",
			CreatedBy: "user-1",
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderRun{
			ID:                 "run-2",
			CreatedAt:          now.Add(-30 * time.Second),
			UpdatedAt:          now.Add(-30 * time.Second),
			SessionID:          "session-2",
			TriggerMessageID:   "message-2",
			Status:             entities.BuilderRunStatusQueued,
			Phase:              &queuedPhase,
			RequestedBy:        "user-1",
			InstructionSummary: "Scoped cancel prompt",
		}).Error)

		r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/builder-sessions/session-1/runs/run-2/cancel", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Error, "builder run run-2 not found")

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", "run-2").Error)
		assert.Nil(t, persistedRun.CancelRequestedAt)
	})
}

func TestStreamBuilderRunLogsReplaysDurableEvents(t *testing.T) {
	setupBuilderSessionHandlerTestDB(t)
	seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)

	now := time.Now().UTC().Truncate(time.Second)
	completedAt := now.Add(-30 * time.Second)
	finalizingPhase := entities.BuilderRunPhaseFinalizing
	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-stream-1",
			CreatedAt: now.Add(-2 * time.Minute),
			UpdatedAt: now.Add(-2 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Replay stream session",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderMessage{
		ID:        "message-stream-1",
		CreatedAt: now.Add(-90 * time.Second),
		UpdatedAt: now.Add(-90 * time.Second),
		SessionID: "session-stream-1",
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Replay durable builder logs",
		CreatedBy: "user-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-stream-1",
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          "session-stream-1",
		TriggerMessageID:   "message-stream-1",
		Status:             entities.BuilderRunStatusSucceeded,
		Phase:              &finalizingPhase,
		RequestedBy:        "user-1",
		InstructionSummary: "Replay durable builder logs",
		ExecutionLog:       "legacy independent execution log",
		CompletedAt:        &completedAt,
	}).Error)
	require.NoError(t, db.DB.Create(&[]entities.BuilderRunEvent{
		{
			ID:        "run-stream-event-1",
			CreatedAt: now.Add(-55 * time.Second),
			RunID:     "run-stream-1",
			Sequence:  1,
			Level:     entities.BuilderRunEventLevelInfo,
			Kind:      entities.BuilderRunEventKindLog,
			Message:   "[system] run started\n",
		},
		{
			ID:        "run-stream-event-2",
			CreatedAt: now.Add(-45 * time.Second),
			RunID:     "run-stream-1",
			Sequence:  2,
			Level:     entities.BuilderRunEventLevelInfo,
			Kind:      entities.BuilderRunEventKindStatus,
			Message:   "[system] run completed\n",
		},
	}).Error)

	r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/session-stream-1/runs/run-stream-1/logs", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")

	body := w.Body.String()
	assert.Contains(t, body, "id:1")
	assert.Contains(t, body, "id:2")
	assert.Contains(t, body, "[system] run started")
	assert.Contains(t, body, "[system] run completed")
	assert.NotContains(t, body, "legacy independent execution log")
	assert.Contains(t, body, "event:done")
	assert.Less(t, strings.Index(body, "[system] run completed"), strings.Index(body, "event:done"))
}

func TestStreamBuilderRunLogsEmitsDoneAfterReplayTurnsTerminal(t *testing.T) {
	setupBuilderSessionHandlerTestDB(t)
	seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)

	now := time.Now().UTC().Truncate(time.Second)
	generatingPhase := entities.BuilderRunPhaseGenerating
	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-stream-race-1",
			CreatedAt: now.Add(-2 * time.Minute),
			UpdatedAt: now.Add(-2 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Replay race session",
		Status:         entities.BuilderSessionStatusRunning,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderMessage{
		ID:        "message-stream-race-1",
		CreatedAt: now.Add(-90 * time.Second),
		UpdatedAt: now.Add(-90 * time.Second),
		SessionID: "session-stream-race-1",
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Replay while the run turns terminal",
		CreatedBy: "user-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-stream-race-1",
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          "session-stream-race-1",
		TriggerMessageID:   "message-stream-race-1",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              &generatingPhase,
		RequestedBy:        "user-1",
		InstructionSummary: "Replay while the run turns terminal",
	}).Error)
	require.NoError(t, db.DB.Create(&[]entities.BuilderRunEvent{
		{
			ID:        "run-stream-race-event-1",
			CreatedAt: now.Add(-40 * time.Second),
			RunID:     "run-stream-race-1",
			Sequence:  1,
			Level:     entities.BuilderRunEventLevelInfo,
			Kind:      entities.BuilderRunEventKindLog,
			Message:   "[system] replayed start\n",
		},
		{
			ID:        "run-stream-race-event-2",
			CreatedAt: now.Add(-30 * time.Second),
			RunID:     "run-stream-race-1",
			Sequence:  2,
			Level:     entities.BuilderRunEventLevelInfo,
			Kind:      entities.BuilderRunEventKindStatus,
			Message:   "[system] replayed completion\n",
		},
	}).Error)

	r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
	requestCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/session-stream-race-1/runs/run-stream-race-1/logs", nil).WithContext(requestCtx)
	w := &builderStreamHookRecorder{ResponseRecorder: httptest.NewRecorder()}
	w.flushHook = func() {
		finalizingPhase := entities.BuilderRunPhaseFinalizing
		completedAt := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", "run-stream-race-1").Updates(map[string]any{
			"status":       entities.BuilderRunStatusSucceeded,
			"phase":        &finalizingPhase,
			"completed_at": &completedAt,
		}).Error)
	}

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "id:1")
	assert.Contains(t, body, "id:2")
	assert.Contains(t, body, "[system] replayed start")
	assert.Contains(t, body, "[system] replayed completion")
	assert.Contains(t, body, "event:done")
	assert.Less(t, strings.Index(body, "[system] replayed completion"), strings.Index(body, "event:done"))
}

func TestStreamBuilderRunLogsDrainsQueuedLiveEventsBeforeDone(t *testing.T) {
	setupBuilderSessionHandlerTestDB(t)
	seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)

	now := time.Now().UTC().Truncate(time.Second)
	generatingPhase := entities.BuilderRunPhaseGenerating
	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-stream-drain-1",
			CreatedAt: now.Add(-2 * time.Minute),
			UpdatedAt: now.Add(-2 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Drain queued live events session",
		Status:         entities.BuilderSessionStatusRunning,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderMessage{
		ID:        "message-stream-drain-1",
		CreatedAt: now.Add(-90 * time.Second),
		UpdatedAt: now.Add(-90 * time.Second),
		SessionID: "session-stream-drain-1",
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Drain queued live events before done",
		CreatedBy: "user-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-stream-drain-1",
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          "session-stream-drain-1",
		TriggerMessageID:   "message-stream-drain-1",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              &generatingPhase,
		RequestedBy:        "user-1",
		InstructionSummary: "Drain queued live events before done",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderRunEvent{
		ID:        "run-stream-drain-event-1",
		CreatedAt: now.Add(-40 * time.Second),
		RunID:     "run-stream-drain-1",
		Sequence:  1,
		Level:     entities.BuilderRunEventLevelInfo,
		Kind:      entities.BuilderRunEventKindLog,
		Message:   "[system] replayed start\n",
	}).Error)

	r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/session-stream-drain-1/runs/run-stream-drain-1/logs", nil)
	w := &builderStreamHookRecorder{ResponseRecorder: httptest.NewRecorder()}
	w.flushHook = func() {
		finalizingPhase := entities.BuilderRunPhaseFinalizing
		completedAt := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", "run-stream-drain-1").Updates(map[string]any{
			"status":       entities.BuilderRunStatusSucceeded,
			"phase":        &finalizingPhase,
			"completed_at": &completedAt,
		}).Error)
		_, err := services.AppendBuilderRunEvent(context.Background(), "run-stream-drain-1", services.BuilderRunEventInput{
			Kind:    entities.BuilderRunEventKindStatus,
			Message: "[system] queued completion\n",
		})
		require.NoError(t, err)
	}

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "id:1")
	assert.Contains(t, body, "[system] replayed start")
	assert.Contains(t, body, "id:2")
	assert.Contains(t, body, "[system] queued completion")
	assert.Contains(t, body, "event:done")
	assert.Less(t, strings.Index(body, "[system] queued completion"), strings.Index(body, "event:done"))
}

func TestStreamBuilderRunLogsEmitsDoneWhenReplayCompletesBeforeTerminalFinalize(t *testing.T) {
	setupBuilderSessionHandlerTestDB(t)
	seedBuilderSessionProjectMember(t, "project-1", "developer-1", app.ProjectRoleDeveloper)
	originalPollInterval := builderRunLogsTerminalPollInterval
	builderRunLogsTerminalPollInterval = 10 * time.Millisecond
	t.Cleanup(func() {
		builderRunLogsTerminalPollInterval = originalPollInterval
	})

	now := time.Now().UTC().Truncate(time.Second)
	generatingPhase := entities.BuilderRunPhaseGenerating
	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-stream-replay-finalize-1",
			CreatedAt: now.Add(-2 * time.Minute),
			UpdatedAt: now.Add(-2 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Replay finalize race session",
		Status:         entities.BuilderSessionStatusRunning,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderMessage{
		ID:        "message-stream-replay-finalize-1",
		CreatedAt: now.Add(-90 * time.Second),
		UpdatedAt: now.Add(-90 * time.Second),
		SessionID: "session-stream-replay-finalize-1",
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Replay a final event before the run turns terminal",
		CreatedBy: "user-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-stream-replay-finalize-1",
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          "session-stream-replay-finalize-1",
		TriggerMessageID:   "message-stream-replay-finalize-1",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              &generatingPhase,
		RequestedBy:        "user-1",
		InstructionSummary: "Replay a final event before the run turns terminal",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderRunEvent{
		ID:        "run-stream-replay-finalize-event-1",
		CreatedAt: now.Add(-20 * time.Second),
		RunID:     "run-stream-replay-finalize-1",
		Sequence:  1,
		Level:     entities.BuilderRunEventLevelInfo,
		Kind:      entities.BuilderRunEventKindStatus,
		Message:   "[system] replayed final event\n",
	}).Error)

	r := newBuilderSessionHandlerRouter("developer-1", "alice", app.UserRoleUser)
	requestCtx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/session-stream-replay-finalize-1/runs/run-stream-replay-finalize-1/logs", nil).WithContext(requestCtx)
	w := &builderStreamHookRecorder{ResponseRecorder: httptest.NewRecorder()}
	w.flushHook = func() {
		time.AfterFunc(30*time.Millisecond, func() {
			finalizingPhase := entities.BuilderRunPhaseFinalizing
			completedAt := time.Now().UTC().Truncate(time.Second)
			_ = db.DB.Model(&entities.BuilderRun{}).Where("id = ?", "run-stream-replay-finalize-1").Updates(map[string]any{
				"status":       entities.BuilderRunStatusSucceeded,
				"phase":        &finalizingPhase,
				"completed_at": &completedAt,
			}).Error
		})
	}

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "id:1")
	assert.Contains(t, body, "[system] replayed final event")
	assert.Contains(t, body, "event:done")
	assert.Less(t, strings.Index(body, "[system] replayed final event"), strings.Index(body, "event:done"))
}

func TestGetBuilderSessionHandler(t *testing.T) {
	t.Run("returns builder session detail for viewers", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.Error)

		var detail models.BuilderSessionDetailResponse
		require.NoError(t, json.Unmarshal(resp.Data, &detail))
		assert.Equal(t, sessionID, detail.Session.ID)
		assert.Equal(t, "Builder session summary", detail.Session.Summary)
		assert.Equal(t, "workspace-2", detail.Session.CurrentWorkspaceID)
		assert.Equal(t, string(entities.BuilderWorkspaceStatusActive), detail.Session.CurrentWorkspaceStatus)
		assert.Equal(t, "run-2", detail.Session.LatestRunID)
		assert.Equal(t, string(entities.BuilderRunStatusSucceeded), detail.Session.LatestRunStatus)
		require.Len(t, detail.Messages, 3)
		assert.Equal(t, []string{"message-1", "message-2", "message-3"}, []string{detail.Messages[0].ID, detail.Messages[1].ID, detail.Messages[2].ID})
		require.Len(t, detail.Runs, 2)
		assert.Equal(t, []string{"run-1", "run-2"}, []string{detail.Runs[0].ID, detail.Runs[1].ID})
		require.NotNil(t, detail.Workspace)
		assert.Equal(t, "workspace-2", detail.Workspace.ID)
		assert.Equal(t, "builder-session-1-v2", detail.Workspace.Namespace)
		require.Len(t, detail.Artifacts, 1)
		assert.Equal(t, "artifact-2", detail.Artifacts[0].ID)
		assert.Equal(t, "plans/new-plan.md", detail.Artifacts[0].Path)
	})

	t.Run("maps missing sessions to not found", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/missing-session", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)

		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Error, "builder session missing-session not found")
	})
}

func TestGetBuilderSessionHandlerReturnsLatestRunArtifactsWithoutActiveWorkspace(t *testing.T) {
	setupBuilderSessionHandlerTestDB(t)
	seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
	sessionID := seedBuilderSessionTerminatedWorkspaceArtifactFixture(t)

	r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp builderSessionAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Empty(t, resp.Error)

	var detail models.BuilderSessionDetailResponse
	require.NoError(t, json.Unmarshal(resp.Data, &detail))
	assert.Nil(t, detail.Workspace)
	require.Len(t, detail.Artifacts, 2)

	artifactsByPath := make(map[string]models.BuilderArtifactSummaryResponse, len(detail.Artifacts))
	for _, artifact := range detail.Artifacts {
		artifactsByPath[artifact.Path] = artifact
	}

	workspaceArtifact, ok := artifactsByPath["src/main.tsx"]
	require.True(t, ok)
	assert.Equal(t, string(entities.BuilderArtifactKindWorkspaceFile), workspaceArtifact.Kind)
	assert.JSONEq(t, `{"size_bytes":120,"source_phase":"materializing_files"}`, workspaceArtifact.MetadataJSON)

	buildOutputArtifact, ok := artifactsByPath["dist/index.html"]
	require.True(t, ok)
	assert.Equal(t, string(entities.BuilderArtifactKindBuildOutput), buildOutputArtifact.Kind)
	assert.JSONEq(t, `{"size_bytes":512,"output_root":"dist","source_phase":"building"}`, buildOutputArtifact.MetadataJSON)
}

func TestGetBuilderSessionPreviewHandler(t *testing.T) {
	t.Run("returns preview summary for viewers", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)

		now := time.Now().UTC().Truncate(time.Second)
		publishedAt := now.Add(-2 * time.Minute)
		completedAt := now.Add(-3 * time.Minute)
		require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", "run-2").Update("completed_at", completedAt).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
			ID:               "snapshot-handler-preview",
			CreatedAt:        publishedAt,
			UpdatedAt:        publishedAt,
			SessionID:        sessionID,
			RunID:            "run-2",
			WorkspaceID:      "workspace-2",
			Status:           entities.BuilderOutputSnapshotStatusPreviewable,
			OutputRoot:       "dist",
			DefaultEntryPath: "dist/index.html",
			StoragePath:      "sessions/handler-preview/snapshot",
			FileCount:        2,
			TotalSizeBytes:   2560,
			PublishedAt:      publishedAt,
		}).Error)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/preview", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.Error)

		var preview models.BuilderSessionPreviewResponse
		require.NoError(t, json.Unmarshal(resp.Data, &preview))
		assert.True(t, preview.Available)
		assert.Equal(t, "previewable", preview.Status)
		assert.Equal(t, "run-2", preview.ResolvedRunID)
		assert.Equal(t, "dist", preview.OutputRoot)
		assert.Equal(t, "dist/index.html", preview.DefaultEntryPath)
		assert.True(t, preview.DownloadAvailable)
		assert.True(t, preview.PreviewAvailable)
		assert.Contains(t, preview.DownloadURL, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/runs/run-2/delivery/download")
		assert.Contains(t, preview.PreviewLaunchURL, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/runs/run-2/preview/launch")
	})
}

func TestLaunchBuilderSessionPreviewHandler(t *testing.T) {
	t.Run("returns frame url for previewable run", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
			ID:               "snapshot-handler-launch",
			CreatedAt:        time.Now().UTC().Add(-2 * time.Minute),
			UpdatedAt:        time.Now().UTC().Add(-2 * time.Minute),
			SessionID:        sessionID,
			RunID:            "run-2",
			WorkspaceID:      "workspace-2",
			Status:           entities.BuilderOutputSnapshotStatusPreviewable,
			OutputRoot:       "dist",
			DefaultEntryPath: "dist/index.html",
			StoragePath:      "sessions/handler-launch/snapshot",
			FileCount:        1,
			TotalSizeBytes:   512,
			PublishedAt:      time.Now().UTC().Add(-2 * time.Minute),
		}).Error)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/runs/run-2/preview/launch", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.Error)

		var launch models.BuilderPreviewLaunchResponse
		require.NoError(t, json.Unmarshal(resp.Data, &launch))
		assert.Contains(t, launch.FrameURL, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/")
		assert.Contains(t, w.Header().Get("Set-Cookie"), services.BuilderPreviewSessionCookieName+"=")
		assert.Contains(t, w.Header().Get("Set-Cookie"), "HttpOnly")
		assert.Contains(t, w.Header().Get("Set-Cookie"), "Secure")
	})
}

func TestReadBuilderPreviewAssetHandler(t *testing.T) {
	t.Run("requires preview session cookie", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects mismatched preview session cookie scope", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		require.NoError(t, db.DB.Create(&entities.User{
			Base:     entities.Base{ID: "viewer-1"},
			Username: "viewer",
			Email:    "viewer@example.com",
			Password: "test-password",
			Fullname: "viewer",
			Role:     app.UserRoleUser,
		}).Error)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)

		token, err := services.MintBuilderPreviewSessionToken("viewer-1", "project-1", sessionID, "run-other")
		require.NoError(t, err)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/", nil)
		req.AddCookie(&http.Cookie{Name: services.BuilderPreviewSessionCookieName, Value: token})
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects expired preview session cookie", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		require.NoError(t, db.DB.Create(&entities.User{
			Base:     entities.Base{ID: "viewer-1"},
			Username: "viewer",
			Email:    "viewer@example.com",
			Password: "test-password",
			Fullname: "viewer",
			Role:     app.UserRoleUser,
		}).Error)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)

		claims := services.BuilderPreviewClaims{
			UserID:    "viewer-1",
			ProjectID: "project-1",
			SessionID: sessionID,
			RunID:     "run-2",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(app.Config.JWTSecret))
		require.NoError(t, err)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/", nil)
		req.AddCookie(&http.Cookie{Name: services.BuilderPreviewSessionCookieName, Value: signed})
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("serves default preview entry when cookie is valid", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		require.NoError(t, db.DB.Create(&entities.User{
			Base:     entities.Base{ID: "viewer-1"},
			Username: "viewer",
			Email:    "viewer@example.com",
			Password: "test-password",
			Fullname: "viewer",
			Role:     app.UserRoleUser,
		}).Error)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)
		baseDir := t.TempDir()
		app.Config.BuilderSnapshotBaseDir = baseDir
		publishedAt := time.Now().UTC().Add(-2 * time.Minute)
		storagePath := "sessions/handler-raw-preview/snapshot"
		require.NoError(t, os.MkdirAll(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist", "index.html"), []byte("<html><body>preview</body></html>"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist", "assets"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist", "assets", "app.js"), []byte("console.log('preview');\n"), 0o644))
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
			ID:               "snapshot-handler-raw-preview",
			CreatedAt:        publishedAt,
			UpdatedAt:        publishedAt,
			SessionID:        sessionID,
			RunID:            "run-2",
			WorkspaceID:      "workspace-2",
			Status:           entities.BuilderOutputSnapshotStatusPreviewable,
			OutputRoot:       "dist",
			DefaultEntryPath: "dist/index.html",
			StoragePath:      storagePath,
			FileCount:        1,
			TotalSizeBytes:   33,
			PublishedAt:      publishedAt,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshotFile{
			ID:             "snapshot-handler-raw-preview-file",
			CreatedAt:      publishedAt,
			UpdatedAt:      publishedAt,
			SnapshotID:     "snapshot-handler-raw-preview",
			RelativePath:   "dist/index.html",
			StoragePath:    storagePath + "/dist/index.html",
			SizeBytes:      33,
			ContentType:    "text/html; charset=utf-8",
			IsDefaultEntry: true,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshotFile{
			ID:             "snapshot-handler-raw-preview-asset",
			CreatedAt:      publishedAt,
			UpdatedAt:      publishedAt,
			SnapshotID:     "snapshot-handler-raw-preview",
			RelativePath:   "dist/assets/app.js",
			StoragePath:    storagePath + "/dist/assets/app.js",
			SizeBytes:      24,
			ContentType:    "application/javascript",
			IsDefaultEntry: false,
		}).Error)

		token, err := services.MintBuilderPreviewSessionToken("viewer-1", "project-1", sessionID, "run-2")
		require.NoError(t, err)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/", nil)
		req.AddCookie(&http.Cookie{Name: services.BuilderPreviewSessionCookieName, Value: token})
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Contains(t, w.Header().Get("Content-Security-Policy"), "sandbox")
		assert.Equal(t, "<html><body>preview</body></html>", w.Body.String())

		assetReq := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/assets/app.js", nil)
		assetReq.AddCookie(&http.Cookie{Name: services.BuilderPreviewSessionCookieName, Value: token})
		assetW := httptest.NewRecorder()

		r.ServeHTTP(assetW, assetReq)

		require.Equal(t, http.StatusOK, assetW.Code)
		assert.Contains(t, assetW.Header().Get("Content-Type"), "application/javascript")
		assert.Equal(t, "console.log('preview');\n", assetW.Body.String())
	})

	t.Run("rejects preview cookie after project membership is removed", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		require.NoError(t, db.DB.Create(&entities.User{
			Base:     entities.Base{ID: "viewer-1"},
			Username: "viewer",
			Email:    "viewer@example.com",
			Password: "test-password",
			Fullname: "viewer",
			Role:     app.UserRoleUser,
		}).Error)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)

		token, err := services.MintBuilderPreviewSessionToken("viewer-1", "project-1", sessionID, "run-2")
		require.NoError(t, err)
		require.NoError(t, db.DB.Where("project_id = ? AND user_id = ?", "project-1", "viewer-1").Delete(&entities.ProjectMember{}).Error)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/", nil)
		req.AddCookie(&http.Cookie{Name: services.BuilderPreviewSessionCookieName, Value: token})
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("allows system admin through preview route without project membership", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		require.NoError(t, db.DB.Create(&entities.User{
			Base:     entities.Base{ID: "admin-1"},
			Username: "admin",
			Email:    "admin@example.com",
			Password: "test-password",
			Fullname: "admin",
			Role:     app.UserRoleAdmin,
		}).Error)
		sessionID := seedBuilderSessionDetailFixture(t)
		baseDir := t.TempDir()
		app.Config.BuilderSnapshotBaseDir = baseDir
		publishedAt := time.Now().UTC().Add(-2 * time.Minute)
		storagePath := "sessions/handler-admin-preview/snapshot"
		require.NoError(t, os.MkdirAll(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist", "index.html"), []byte("<html><body>admin preview</body></html>"), 0o644))
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
			ID:               "snapshot-handler-admin-preview",
			CreatedAt:        publishedAt,
			UpdatedAt:        publishedAt,
			SessionID:        sessionID,
			RunID:            "run-2",
			WorkspaceID:      "workspace-2",
			Status:           entities.BuilderOutputSnapshotStatusPreviewable,
			OutputRoot:       "dist",
			DefaultEntryPath: "dist/index.html",
			StoragePath:      storagePath,
			FileCount:        1,
			TotalSizeBytes:   39,
			PublishedAt:      publishedAt,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshotFile{
			ID:             "snapshot-handler-admin-preview-file",
			CreatedAt:      publishedAt,
			UpdatedAt:      publishedAt,
			SnapshotID:     "snapshot-handler-admin-preview",
			RelativePath:   "dist/index.html",
			StoragePath:    storagePath + "/dist/index.html",
			SizeBytes:      39,
			ContentType:    "text/html; charset=utf-8",
			IsDefaultEntry: true,
		}).Error)

		token, err := services.MintBuilderPreviewSessionToken("admin-1", "project-1", sessionID, "run-2")
		require.NoError(t, err)

		r := newBuilderSessionHandlerRouter("admin-1", "admin", app.UserRoleAdmin)
		req := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/", nil)
		req.AddCookie(&http.Cookie{Name: services.BuilderPreviewSessionCookieName, Value: token})
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "<html><body>admin preview</body></html>", w.Body.String())
	})

	t.Run("maps malformed preview asset paths to bad request", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		require.NoError(t, db.DB.Create(&entities.User{
			Base:     entities.Base{ID: "viewer-1"},
			Username: "viewer",
			Email:    "viewer@example.com",
			Password: "test-password",
			Fullname: "viewer",
			Role:     app.UserRoleUser,
		}).Error)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)
		baseDir := t.TempDir()
		app.Config.BuilderSnapshotBaseDir = baseDir
		publishedAt := time.Now().UTC().Add(-2 * time.Minute)
		storagePath := "sessions/handler-malformed-preview/snapshot"
		require.NoError(t, os.MkdirAll(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist", "index.html"), []byte("<html><body>preview</body></html>"), 0o644))
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
			ID:               "snapshot-handler-malformed-preview",
			CreatedAt:        publishedAt,
			UpdatedAt:        publishedAt,
			SessionID:        sessionID,
			RunID:            "run-2",
			WorkspaceID:      "workspace-2",
			Status:           entities.BuilderOutputSnapshotStatusPreviewable,
			OutputRoot:       "dist",
			DefaultEntryPath: "dist/index.html",
			StoragePath:      storagePath,
			FileCount:        1,
			TotalSizeBytes:   33,
			PublishedAt:      publishedAt,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshotFile{
			ID:             "snapshot-handler-malformed-preview-file",
			CreatedAt:      publishedAt,
			UpdatedAt:      publishedAt,
			SnapshotID:     "snapshot-handler-malformed-preview",
			RelativePath:   "dist/index.html",
			StoragePath:    storagePath + "/dist/index.html",
			SizeBytes:      33,
			ContentType:    "text/html; charset=utf-8",
			IsDefaultEntry: true,
		}).Error)
		token, err := services.MintBuilderPreviewSessionToken("viewer-1", "project-1", sessionID, "run-2")
		require.NoError(t, err)

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-2/%2E%2E/%2E%2E/secret.txt", nil)
		req.AddCookie(&http.Cookie{Name: services.BuilderPreviewSessionCookieName, Value: token})
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDownloadBuilderSessionSnapshotHandler(t *testing.T) {
	t.Run("streams snapshot archive for viewers", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)
		publishedAt := time.Now().UTC().Add(-2 * time.Minute)
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
			ID:               "snapshot-handler-download",
			CreatedAt:        publishedAt,
			UpdatedAt:        publishedAt,
			SessionID:        sessionID,
			RunID:            "run-2",
			WorkspaceID:      "workspace-2",
			Status:           entities.BuilderOutputSnapshotStatusPreviewable,
			OutputRoot:       "dist",
			DefaultEntryPath: "dist/index.html",
			StoragePath:      "sessions/handler-download/snapshot",
			FileCount:        1,
			TotalSizeBytes:   512,
			PublishedAt:      publishedAt,
		}).Error)

		originalWriteArchive := writeBuilderSessionSnapshotArchive
		writeBuilderSessionSnapshotArchive = func(ctx context.Context, projectID, sessionID, runID string, writer io.Writer) error {
			_, err := writer.Write([]byte("snapshot-archive"))
			return err
		}
		defer func() { writeBuilderSessionSnapshotArchive = originalWriteArchive }()

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/runs/run-2/delivery/download", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/gzip", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Header().Get("Content-Disposition"), "builder-output-run-2.tar.gz")
		assert.Equal(t, "snapshot-archive", w.Body.String())
	})

	t.Run("maps snapshot download failures to builder session error responses", func(t *testing.T) {
		setupBuilderSessionHandlerTestDB(t)
		seedBuilderSessionProjectMember(t, "project-1", "viewer-1", app.ProjectRoleViewer)
		sessionID := seedBuilderSessionDetailFixture(t)

		originalWriteArchive := writeBuilderSessionSnapshotArchive
		writeBuilderSessionSnapshotArchive = func(ctx context.Context, projectID, sessionID, runID string, writer io.Writer) error {
			return &services.BuilderRunNotFoundError{ProjectID: projectID, SessionID: sessionID, RunID: runID}
		}
		defer func() { writeBuilderSessionSnapshotArchive = originalWriteArchive }()

		r := newBuilderSessionHandlerRouter("viewer-1", "viewer", app.UserRoleUser)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/runs/run-missing/delivery/download", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		var resp builderSessionAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Error, "builder run run-missing not found")
	})
}
