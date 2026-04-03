package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/routes"
	"github.com/ketches/ketches/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type builderSessionRouteAPIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func setupBuilderSessionRoutesTestDB(t *testing.T) {
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

	db.DB = testDB
	require.NoError(t, db.Migrate())
	app.Config.JWTSecret = "builder-session-routes-test-secret"
}

func seedBuilderSessionRouteUser(t *testing.T, userID, username, role string) *entities.User {
	t.Helper()

	user := &entities.User{
		Base:     entities.Base{ID: userID},
		Username: username,
		Email:    username + "@example.com",
		Password: "test-password",
		Fullname: username,
		Role:     role,
	}
	require.NoError(t, db.DB.Create(user).Error)
	return user
}

func seedBuilderSessionRouteProjectMember(t *testing.T, projectID, userID, projectRole string) {
	t.Helper()

	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          projectID + "-" + userID,
		ProjectID:   projectID,
		UserID:      userID,
		ProjectRole: projectRole,
	}).Error)
}

func seedBuilderSessionRouteReadFixture(t *testing.T) string {
	t.Helper()

	now := time.Now().UTC()
	session := entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-route-1",
			CreatedAt: now.Add(-5 * time.Minute),
			UpdatedAt: now.Add(-4 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Route smoke session",
		Summary:        "Existing builder session",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "developer-1",
		LastActivityAt: now.Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(&session).Error)

	message := entities.BuilderMessage{
		ID:        "message-route-1",
		CreatedAt: now.Add(-3 * time.Minute),
		UpdatedAt: now.Add(-3 * time.Minute),
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Initial route smoke prompt",
		CreatedBy: "developer-1",
	}
	require.NoError(t, db.DB.Create(&message).Error)

	run := entities.BuilderRun{
		ID:                 "run-route-1",
		CreatedAt:          now.Add(-2 * time.Minute),
		UpdatedAt:          now.Add(-2 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   message.ID,
		Status:             entities.BuilderRunStatusQueued,
		RequestedBy:        "developer-1",
		InstructionSummary: message.Content,
	}
	require.NoError(t, db.DB.Create(&run).Error)

	return session.ID
}

func seedBuilderSessionRouteBuildOutputFixture(t *testing.T) string {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	terminatedAt := now.Add(-30 * time.Second)
	workspaceID := "workspace-route-build-output"
	sessionID := "session-route-build-output"
	runID := "run-route-build-output"

	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base: entities.Base{
			ID:        sessionID,
			CreatedAt: now.Add(-5 * time.Minute),
			UpdatedAt: now.Add(-4 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Route build output session",
		Summary:        "Build outputs should remain visible on the existing detail route.",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "developer-1",
		LastActivityAt: now.Add(-time.Minute),
	}).Error)

	require.NoError(t, db.DB.Create(&entities.BuilderMessage{
		ID:        "message-route-build-output",
		CreatedAt: now.Add(-4 * time.Minute),
		UpdatedAt: now.Add(-4 * time.Minute),
		SessionID: sessionID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Build the generated app.",
		CreatedBy: "developer-1",
	}).Error)

	completedAt := now.Add(-2 * time.Minute)
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 runID,
		CreatedAt:          now.Add(-3 * time.Minute),
		UpdatedAt:          now.Add(-2 * time.Minute),
		SessionID:          sessionID,
		TriggerMessageID:   "message-route-build-output",
		WorkspaceID:        &workspaceID,
		Status:             entities.BuilderRunStatusSucceeded,
		RequestedBy:        "developer-1",
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
		Namespace:     "builder-route-build-output",
		PodName:       "builder-route-build-output-0",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusExpired,
		WorkspaceRoot: "/workspace/route-build-output",
		TerminatedAt:  &terminatedAt,
	}).Error)

	require.NoError(t, db.DB.Create(&[]entities.BuilderArtifact{
		{
			ID:           "artifact-route-build-output-source",
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
			ID:           "artifact-route-build-output-build",
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

func newBuilderSessionRouteRequest(t *testing.T, method, path, body string, user *entities.User) *http.Request {
	t.Helper()

	var bodyReader *strings.Reader
	if body == "" {
		bodyReader = strings.NewReader("")
	} else {
		bodyReader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != nil {
		token, err := app.GenerateAccessToken(user)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions && method != http.MethodTrace {
			req.AddCookie(&http.Cookie{Name: app.CSRFCookieName, Value: "builder-session-route-csrf"})
			req.Header.Set(app.CSRFHeaderName, "builder-session-route-csrf")
		}
	}
	return req
}

func TestBuilderSessionRoutes(t *testing.T) {
	setupBuilderSessionRoutesTestDB(t)

	viewer := seedBuilderSessionRouteUser(t, "viewer-1", "viewer", app.UserRoleUser)
	developer := seedBuilderSessionRouteUser(t, "developer-1", "developer", app.UserRoleUser)
	seedBuilderSessionRouteProjectMember(t, "project-1", viewer.ID, app.ProjectRoleViewer)
	seedBuilderSessionRouteProjectMember(t, "project-1", developer.ID, app.ProjectRoleDeveloper)
	existingSessionID := seedBuilderSessionRouteReadFixture(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.SetupRoutes(r)

	t.Run("requires auth before builder handlers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/builder-sessions", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("uses production read registrations with viewer access", func(t *testing.T) {
		listReq := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions", "", viewer)
		listW := httptest.NewRecorder()
		r.ServeHTTP(listW, listReq)
		require.Equal(t, http.StatusOK, listW.Code)

		var listResp builderSessionRouteAPIResponse
		require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listResp))
		var items []models.BuilderSessionListItem
		require.NoError(t, json.Unmarshal(listResp.Data, &items))
		require.NotEmpty(t, items)

		detailReq := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+existingSessionID, "", viewer)
		detailW := httptest.NewRecorder()
		r.ServeHTTP(detailW, detailReq)
		require.Equal(t, http.StatusOK, detailW.Code)

		var detailResp builderSessionRouteAPIResponse
		require.NoError(t, json.Unmarshal(detailW.Body.Bytes(), &detailResp))
		var detail models.BuilderSessionDetailResponse
		require.NoError(t, json.Unmarshal(detailResp.Data, &detail))
		assert.Equal(t, existingSessionID, detail.Session.ID)

		viewerCreateReq := newBuilderSessionRouteRequest(t, http.MethodPost, "/api/v1/projects/project-1/builder-sessions", `{"build_env_id":"env-1","prompt":"Viewer should be blocked."}`, viewer)
		viewerCreateW := httptest.NewRecorder()
		r.ServeHTTP(viewerCreateW, viewerCreateReq)
		require.Equal(t, http.StatusForbidden, viewerCreateW.Code)
	})

	t.Run("uses production write registrations with developer access", func(t *testing.T) {
		createReq := newBuilderSessionRouteRequest(t, http.MethodPost, "/api/v1/projects/project-1/builder-sessions", `{"build_env_id":"env-1","title":"Route-created session","prompt":"Create the route integration session."}`, developer)
		createW := httptest.NewRecorder()
		r.ServeHTTP(createW, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		var createResp builderSessionRouteAPIResponse
		require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &createResp))
		var created models.BuilderSessionDetailResponse
		require.NoError(t, json.Unmarshal(createResp.Data, &created))
		require.NotEmpty(t, created.Session.ID)
		require.NotEmpty(t, created.Session.LatestRunID)

		finalizingPhase := entities.BuilderRunPhaseFinalizing
		require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", created.Session.LatestRunID).Updates(map[string]any{
			"status":        entities.BuilderRunStatusSucceeded,
			"phase":         &finalizingPhase,
			"execution_log": "legacy route execution log",
			"completed_at":  time.Now().UTC().Truncate(time.Second),
		}).Error)
		require.NoError(t, db.DB.Create(&[]entities.BuilderRunEvent{
			{
				ID:        "route-run-event-1",
				CreatedAt: time.Now().UTC().Add(-2 * time.Second),
				RunID:     created.Session.LatestRunID,
				Sequence:  1,
				Level:     entities.BuilderRunEventLevelInfo,
				Kind:      entities.BuilderRunEventKindLog,
				Message:   "[system] routed replay\n",
			},
			{
				ID:        "route-run-event-2",
				CreatedAt: time.Now().UTC().Add(-time.Second),
				RunID:     created.Session.LatestRunID,
				Sequence:  2,
				Level:     entities.BuilderRunEventLevelInfo,
				Kind:      entities.BuilderRunEventKindStatus,
				Message:   "[system] routed completion\n",
			},
		}).Error)

		logsReq := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+created.Session.ID+"/runs/"+created.Session.LatestRunID+"/logs", "", developer)
		logsW := httptest.NewRecorder()
		r.ServeHTTP(logsW, logsReq)
		require.Equal(t, http.StatusOK, logsW.Code)
		assert.Contains(t, logsW.Body.String(), "id:1")
		assert.Contains(t, logsW.Body.String(), "id:2")
		assert.Contains(t, logsW.Body.String(), "[system] routed replay")
		assert.Contains(t, logsW.Body.String(), "[system] routed completion")
		assert.NotContains(t, logsW.Body.String(), "legacy route execution log")

		resumedLogsReq := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+created.Session.ID+"/runs/"+created.Session.LatestRunID+"/logs?after=1", "", developer)
		resumedLogsW := httptest.NewRecorder()
		r.ServeHTTP(resumedLogsW, resumedLogsReq)
		require.Equal(t, http.StatusOK, resumedLogsW.Code)
		assert.NotContains(t, resumedLogsW.Body.String(), "[system] routed replay")
		assert.Contains(t, resumedLogsW.Body.String(), "id:2")
		assert.Contains(t, resumedLogsW.Body.String(), "[system] routed completion")

		messageReq := newBuilderSessionRouteRequest(t, http.MethodPost, "/api/v1/projects/project-1/builder-sessions/"+created.Session.ID+"/messages", `{"content":"Follow up through the production route."}`, developer)
		messageW := httptest.NewRecorder()
		r.ServeHTTP(messageW, messageReq)
		require.Equal(t, http.StatusCreated, messageW.Code)

		cancelReq := newBuilderSessionRouteRequest(t, http.MethodPost, "/api/v1/projects/project-1/builder-sessions/"+existingSessionID+"/runs/run-route-1/cancel", "", developer)
		cancelW := httptest.NewRecorder()
		r.ServeHTTP(cancelW, cancelReq)
		require.Equal(t, http.StatusOK, cancelW.Code)

		var cancelResp builderSessionRouteAPIResponse
		require.NoError(t, json.Unmarshal(cancelW.Body.Bytes(), &cancelResp))
		var cancelDetail models.BuilderSessionDetailResponse
		require.NoError(t, json.Unmarshal(cancelResp.Data, &cancelDetail))
		assert.Equal(t, existingSessionID, cancelDetail.Session.ID)
		assert.Equal(t, string(entities.BuilderSessionStatusReady), cancelDetail.Session.Status)
		require.Len(t, cancelDetail.Runs, 1)
		assert.Equal(t, string(entities.BuilderRunStatusQueued), cancelDetail.Runs[0].Status)

		var cancelledRun entities.BuilderRun
		require.NoError(t, db.DB.First(&cancelledRun, "id = ?", "run-route-1").Error)
		require.NotNil(t, cancelledRun.CancelRequestedAt)

		now := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-route-2", CreatedAt: now, UpdatedAt: now},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Scoped route session",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      developer.ID,
			LastActivityAt: now,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderMessage{
			ID:        "message-route-2",
			CreatedAt: now,
			UpdatedAt: now,
			SessionID: "session-route-2",
			Role:      entities.BuilderMessageRoleUser,
			Content:   "Scoped route prompt",
			CreatedBy: developer.ID,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderRun{
			ID:                 "run-route-2",
			CreatedAt:          now,
			UpdatedAt:          now,
			SessionID:          "session-route-2",
			TriggerMessageID:   "message-route-2",
			Status:             entities.BuilderRunStatusQueued,
			RequestedBy:        developer.ID,
			InstructionSummary: "Scoped route prompt",
		}).Error)

		wrongScopeReq := newBuilderSessionRouteRequest(t, http.MethodPost, "/api/v1/projects/project-1/builder-sessions/"+existingSessionID+"/runs/run-route-2/cancel", "", developer)
		wrongScopeW := httptest.NewRecorder()
		r.ServeHTTP(wrongScopeW, wrongScopeReq)
		require.Equal(t, http.StatusNotFound, wrongScopeW.Code)

		var scopedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&scopedRun, "id = ?", "run-route-2").Error)
		assert.Nil(t, scopedRun.CancelRequestedAt)
	})
}

func TestBuilderSessionRoutesExposeBuildOutputArtifactsWithoutRouteChurn(t *testing.T) {
	setupBuilderSessionRoutesTestDB(t)

	viewer := seedBuilderSessionRouteUser(t, "viewer-1", "viewer", app.UserRoleUser)
	seedBuilderSessionRouteProjectMember(t, "project-1", viewer.ID, app.ProjectRoleViewer)
	sessionID := seedBuilderSessionRouteBuildOutputFixture(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.SetupRoutes(r)

	req := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID, "", viewer)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp builderSessionRouteAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	var detail models.BuilderSessionDetailResponse
	require.NoError(t, json.Unmarshal(resp.Data, &detail))
	assert.Equal(t, sessionID, detail.Session.ID)
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

func TestBuilderSessionRoutesExposePreviewReadEndpoints(t *testing.T) {
	setupBuilderSessionRoutesTestDB(t)

	viewer := seedBuilderSessionRouteUser(t, "viewer-1", "viewer", app.UserRoleUser)
	seedBuilderSessionRouteProjectMember(t, "project-1", viewer.ID, app.ProjectRoleViewer)
	sessionID := seedBuilderSessionRouteReadFixture(t)
	baseDir := t.TempDir()
	app.Config.BuilderSnapshotBaseDir = baseDir
	publishedAt := time.Now().UTC().Add(-2 * time.Minute)
	storagePath := "sessions/route-preview/snapshot"
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, filepath.FromSlash(storagePath), "dist", "index.html"), []byte("<html><body>route preview</body></html>"), 0o644))
	require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", "run-route-1").Updates(map[string]any{
		"status":       entities.BuilderRunStatusSucceeded,
		"completed_at": time.Now().UTC().Add(-3 * time.Minute),
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
		ID:               "snapshot-route-preview",
		CreatedAt:        publishedAt,
		UpdatedAt:        publishedAt,
		SessionID:        sessionID,
		RunID:            "run-route-1",
		WorkspaceID:      "",
		Status:           entities.BuilderOutputSnapshotStatusPreviewable,
		OutputRoot:       "dist",
		DefaultEntryPath: "dist/index.html",
		StoragePath:      storagePath,
		FileCount:        1,
		TotalSizeBytes:   512,
		PublishedAt:      publishedAt,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshotFile{
		ID:             "snapshot-route-preview-file",
		CreatedAt:      publishedAt,
		UpdatedAt:      publishedAt,
		SnapshotID:     "snapshot-route-preview",
		RelativePath:   "dist/index.html",
		StoragePath:    storagePath + "/dist/index.html",
		SizeBytes:      39,
		ContentType:    "text/html; charset=utf-8",
		IsDefaultEntry: true,
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.SetupRoutes(r)

	previewReq := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/preview", "", viewer)
	previewW := httptest.NewRecorder()
	r.ServeHTTP(previewW, previewReq)
	require.Equal(t, http.StatusOK, previewW.Code)

	launchReq := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/runs/run-route-1/preview/launch", "", viewer)
	launchW := httptest.NewRecorder()
	r.ServeHTTP(launchW, launchReq)
	require.Equal(t, http.StatusOK, launchW.Code)
	launchCookie := launchW.Header().Get("Set-Cookie")
	assert.Contains(t, launchCookie, services.BuilderPreviewSessionCookieName+"=")

	downloadReq := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/runs/run-route-1/delivery/download", "", viewer)
	downloadW := httptest.NewRecorder()
	r.ServeHTTP(downloadW, downloadReq)
	require.Equal(t, http.StatusOK, downloadW.Code)

	rawPreviewReq := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-route-1/", nil)
	rawPreviewReq.Header.Set("Cookie", launchCookie)
	rawPreviewW := httptest.NewRecorder()
	r.ServeHTTP(rawPreviewW, rawPreviewReq)
	require.Equal(t, http.StatusOK, rawPreviewW.Code)
	assert.Contains(t, rawPreviewW.Header().Get("Content-Type"), "text/html")
	assert.Equal(t, "<html><body>route preview</body></html>", rawPreviewW.Body.String())

	missingCookieReq := httptest.NewRequest(http.MethodGet, "/builder-preview/projects/project-1/sessions/"+sessionID+"/runs/run-route-1/", nil)
	missingCookieW := httptest.NewRecorder()
	r.ServeHTTP(missingCookieW, missingCookieReq)
	require.Equal(t, http.StatusUnauthorized, missingCookieW.Code)

	viewerWorkspaceDownloadReq := newBuilderSessionRouteRequest(t, http.MethodGet, "/api/v1/projects/project-1/builder-sessions/"+sessionID+"/files/download", "", viewer)
	viewerWorkspaceDownloadW := httptest.NewRecorder()
	r.ServeHTTP(viewerWorkspaceDownloadW, viewerWorkspaceDownloadReq)
	require.Equal(t, http.StatusForbidden, viewerWorkspaceDownloadW.Code)
}
