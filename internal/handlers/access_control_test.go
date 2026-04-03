package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type accessControlAPIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func setupAccessControlHandlerDB(t *testing.T) {
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
		&entities.User{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Cluster{},
		&entities.ClusterIntegration{},
		&entities.Env{},
	))

	db.DB = testDB
}

func accessControlClaimsMiddleware(userID, username, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID:   userID,
			Username: username,
			Role:     role,
		})
		c.Next()
	}
}

func seedAccessControlUser(t *testing.T, id, username, role string) {
	t.Helper()

	require.NoError(t, db.DB.Create(&entities.User{
		Base:     entities.Base{ID: id},
		Username: username,
		Email:    username + "@example.com",
		Password: "hashed-password",
		Role:     role,
		Fullname: username,
	}).Error)
}

func seedAccessControlProjectCluster(t *testing.T, projectID, clusterID, envID string) {
	t.Helper()

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: projectID},
		Slug: projectID,
		Name: projectID,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: clusterID},
		Slug:       clusterID,
		Name:       clusterID,
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: envID},
		Slug:      envID,
		Name:      envID,
		ProjectID: projectID,
		ClusterID: clusterID,
	}).Error)
}

func seedAccessControlMember(t *testing.T, projectID, userID, role string) {
	t.Helper()

	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          projectID + "-" + userID,
		ProjectID:   projectID,
		UserID:      userID,
		ProjectRole: role,
	}).Error)
}

func TestListUsersRejectsNonAdmin(t *testing.T) {
	setupAccessControlHandlerDB(t)
	seedAccessControlUser(t, "user-1", "alice", app.UserRoleUser)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	r.GET("/api/v1/users", middlewares.AdminOnly(), ListUsers)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateUserRejectsNonAdmin(t *testing.T) {
	setupAccessControlHandlerDB(t)
	seedAccessControlUser(t, "user-1", "alice", app.UserRoleUser)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("user-2", "bob", app.UserRoleUser))
	r.PUT("/api/v1/users/:userID", middlewares.AdminOnly(), UpdateUser)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user-1", strings.NewReader(`{"fullname":"Changed"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)

	var persisted entities.User
	require.NoError(t, db.DB.First(&persisted, "id = ?", "user-1").Error)
	assert.Equal(t, "alice", persisted.Fullname)
}

func TestGetDashboardStatsRejectsNonMemberProject(t *testing.T) {
	setupAccessControlHandlerDB(t)
	seedAccessControlProjectCluster(t, "project-1", "cluster-1", "env-1")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	r.GET("/api/v1/dashboard/stats", GetDashboardStats)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats?project_id=project-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetDashboardEnvironmentsAllowsProjectMember(t *testing.T) {
	setupAccessControlHandlerDB(t)
	seedAccessControlProjectCluster(t, "project-1", "cluster-1", "env-1")
	seedAccessControlMember(t, "project-1", "user-1", app.ProjectRoleDeveloper)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	r.GET("/api/v1/dashboard/environments", GetDashboardEnvironments)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/environments?project_id=project-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetPublicClusterRequiresProjectScopeForNonAdmin(t *testing.T) {
	setupAccessControlHandlerDB(t)
	seedAccessControlProjectCluster(t, "project-1", "cluster-1", "env-1")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	r.GET("/api/v1/clusters/:clusterID/public", GetPublicCluster)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/cluster-1/public", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPublicClusterAllowsAdminWithoutProjectScope(t *testing.T) {
	setupAccessControlHandlerDB(t)
	seedAccessControlProjectCluster(t, "project-1", "cluster-1", "env-1")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("admin-1", "admin", app.UserRoleAdmin))
	r.GET("/api/v1/clusters/:clusterID/public", GetPublicCluster)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/cluster-1/public", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetPublicClusterAllowsProjectMember(t *testing.T) {
	setupAccessControlHandlerDB(t)
	seedAccessControlProjectCluster(t, "project-1", "cluster-1", "env-1")
	seedAccessControlMember(t, "project-1", "user-1", app.ProjectRoleViewer)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	r.GET("/api/v1/clusters/:clusterID/public", GetPublicCluster)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/cluster-1/public?project_id=project-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp accessControlAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Empty(t, resp.Error)
}

func TestProxyPrometheusQueryRejectsMissingProjectScopeForNonAdmin(t *testing.T) {
	setupAccessControlHandlerDB(t)
	seedAccessControlProjectCluster(t, "project-1", "cluster-1", "env-1")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	r.GET("/api/v1/clusters/:clusterID/prometheus/query", ProxyPrometheusQuery)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/cluster-1/prometheus/query?query=up", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
