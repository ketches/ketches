package handlers_test

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
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/routes"
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

		messageReq := newBuilderSessionRouteRequest(t, http.MethodPost, "/api/v1/projects/project-1/builder-sessions/"+created.Session.ID+"/messages", `{"content":"Follow up through the production route."}`, developer)
		messageW := httptest.NewRecorder()
		r.ServeHTTP(messageW, messageReq)
		require.Equal(t, http.StatusCreated, messageW.Code)
	})
}
