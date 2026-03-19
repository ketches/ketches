package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
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

func setupBuilderSessionHandlerTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

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
	projectsRead.GET("/:projectID/builder-sessions", ListBuilderSessions)
	projectsRead.GET("/:projectID/builder-sessions/:sessionID", GetBuilderSession)

	projectsWrite := r.Group("/api/v1/projects", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
	projectsWrite.POST("/:projectID/builder-sessions", CreateBuilderSession)
	projectsWrite.POST("/:projectID/builder-sessions/:sessionID/messages", PostBuilderSessionMessage)

	return r
}

func newBuilderSessionDirectWriteRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/v1/projects/:projectID/builder-sessions", CreateBuilderSession)
	r.POST("/api/v1/projects/:projectID/builder-sessions/:sessionID/messages", PostBuilderSessionMessage)

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
		assert.Equal(t, string(entities.BuilderSessionStatusRunning), detail.Session.Status)
		assert.Equal(t, string(entities.BuilderRunStatusExecuting), detail.Session.LatestRunStatus)
		require.Len(t, detail.Messages, 1)
		assert.Equal(t, "Add pagination to the service list.", detail.Messages[0].Content)
		require.Len(t, detail.Runs, 1)
		assert.Equal(t, string(entities.BuilderRunStatusExecuting), detail.Runs[0].Status)
		assert.NotNil(t, detail.Runs[0].StartedAt)
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
