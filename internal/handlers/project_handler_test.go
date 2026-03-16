package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupProjectHandlerTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.User{},
		&entities.Project{},
		&entities.ProjectMember{},
	))
	db.DB = testDB
}

func projectClaimsMiddleware(userID, username, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID:   userID,
			Username: username,
			Role:     role,
		})
		c.Next()
	}
}

type projectAPIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func TestCreateAndGetProject_ContainsCollaborationEnabled(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(projectClaimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects", CreateProject)
	r.GET("/api/v1/projects/:projectID", GetProject)

	body := `{"name":"Project A","slug":"project-a","description":"desc","collaboration_enabled":true}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	require.Equal(t, http.StatusCreated, createW.Code)
	var createResp projectAPIResponse
	require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &createResp))
	require.Empty(t, createResp.Error)

	var created map[string]any
	require.NoError(t, json.Unmarshal(createResp.Data, &created))
	projectID, ok := created["id"].(string)
	require.True(t, ok)
	assert.Equal(t, true, created["collaboration_enabled"])

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	require.Equal(t, http.StatusOK, getW.Code)
	var getResp projectAPIResponse
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &getResp))
	require.Empty(t, getResp.Error)

	var got map[string]any
	require.NoError(t, json.Unmarshal(getResp.Data, &got))
	assert.Equal(t, true, got["collaboration_enabled"])
}

func TestUpdateProject_UpdatesCollaborationEnabled(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base:                 entities.Base{ID: "p1"},
		Slug:                 "project-b",
		Name:                 "Project B",
		Description:          "desc",
		CollaborationEnabled: true,
	}).Error)

	r := gin.New()
	r.Use(projectClaimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID", UpdateProject)

	updateBody := `{"name":"Project B","description":"updated","collaboration_enabled":false}`
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1", strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)

	require.Equal(t, http.StatusOK, updateW.Code)
	var updateResp projectAPIResponse
	require.NoError(t, json.Unmarshal(updateW.Body.Bytes(), &updateResp))
	require.Empty(t, updateResp.Error)

	var updated map[string]any
	require.NoError(t, json.Unmarshal(updateResp.Data, &updated))
	assert.Equal(t, false, updated["collaboration_enabled"])

	var project entities.Project
	require.NoError(t, db.DB.First(&project, "id = ?", "p1").Error)
	assert.False(t, project.CollaborationEnabled)
}
