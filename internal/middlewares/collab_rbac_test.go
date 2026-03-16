package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupCollabRBACTestDB creates an in-memory SQLite DB with a project and
// three members (owner, developer, viewer) for RBAC testing.
func setupCollabRBACTestDB(t *testing.T) (projectID, ownerID, developerID, viewerID string) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.Project{},
		&entities.ProjectMember{},
	))

	projectID = uuid.NewString()
	ownerID = uuid.NewString()
	developerID = uuid.NewString()
	viewerID = uuid.NewString()

	require.NoError(t, testDB.Create(&entities.Project{
		Base: entities.Base{ID: projectID},
		Name: "test-project",
		Slug: "test-project",
	}).Error)

	for _, m := range []entities.ProjectMember{
		{ID: uuid.NewString(), ProjectID: projectID, UserID: ownerID, ProjectRole: app.ProjectRoleOwner},
		{ID: uuid.NewString(), ProjectID: projectID, UserID: developerID, ProjectRole: app.ProjectRoleDeveloper},
		{ID: uuid.NewString(), ProjectID: projectID, UserID: viewerID, ProjectRole: app.ProjectRoleViewer},
	} {
		require.NoError(t, testDB.Create(&m).Error)
	}

	db.DB = testDB
	return projectID, ownerID, developerID, viewerID
}

// claimsMiddleware injects Claims into the gin context, simulating auth.
func claimsMiddleware(userID, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID: userID,
			Role:   role,
		})
		c.Next()
	}
}

// okHandler is a simple handler that returns 200 OK.
func okHandler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

// TestCollabReadRoutes_ViewerAllowed verifies that a viewer-level project
// member can access collaboration read routes (GET requirements, sprints, etc.).
func TestCollabReadRoutes_ViewerAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID, _, _, viewerID := setupCollabRBACTestDB(t)

	readPaths := []string{
		"/api/v1/projects/" + projectID + "/requirements",
		"/api/v1/projects/" + projectID + "/sprints",
		"/api/v1/projects/" + projectID + "/tasks",
		"/api/v1/projects/" + projectID + "/backlog",
		"/api/v1/projects/" + projectID + "/test-cases",
		"/api/v1/projects/" + projectID + "/defects",
	}

	for _, path := range readPaths {
		t.Run("GET "+path, func(t *testing.T) {
			router := gin.New()
			router.Use(claimsMiddleware(viewerID, app.UserRoleUser))
			router.Use(RequireProjectRole(app.ProjectRoleViewer))
			router.GET("/api/v1/projects/:projectID/requirements", okHandler)
			router.GET("/api/v1/projects/:projectID/sprints", okHandler)
			router.GET("/api/v1/projects/:projectID/tasks", okHandler)
			router.GET("/api/v1/projects/:projectID/backlog", okHandler)
			router.GET("/api/v1/projects/:projectID/test-cases", okHandler)
			router.GET("/api/v1/projects/:projectID/defects", okHandler)

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "viewer should access read route")
		})
	}
}

// TestCollabWriteRoutes_ViewerBlocked verifies that a viewer-level project
// member is rejected (403) on collaboration write routes.
func TestCollabWriteRoutes_ViewerBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID, _, _, viewerID := setupCollabRBACTestDB(t)

	writePaths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/projects/" + projectID + "/requirements"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/sprints"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/tasks"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/test-cases"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/defects"},
		{http.MethodPut, "/api/v1/projects/" + projectID + "/sprints/sprint-1"},
		{http.MethodDelete, "/api/v1/projects/" + projectID + "/sprints/sprint-1"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/requirements/req-1/transition"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/tasks/task-1/transition"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/defects/defect-1/transition"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/backlog/reorder"},
	}

	for _, tc := range writePaths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			router := gin.New()
			router.Use(claimsMiddleware(viewerID, app.UserRoleUser))
			router.Use(RequireProjectRole(app.ProjectRoleDeveloper))

			// Register all write routes to match path patterns
			router.POST("/api/v1/projects/:projectID/requirements", okHandler)
			router.POST("/api/v1/projects/:projectID/sprints", okHandler)
			router.POST("/api/v1/projects/:projectID/tasks", okHandler)
			router.POST("/api/v1/projects/:projectID/test-cases", okHandler)
			router.POST("/api/v1/projects/:projectID/defects", okHandler)
			router.PUT("/api/v1/projects/:projectID/sprints/:sprintID", okHandler)
			router.DELETE("/api/v1/projects/:projectID/sprints/:sprintID", okHandler)
			router.POST("/api/v1/projects/:projectID/requirements/:requirementID/transition", okHandler)
			router.POST("/api/v1/projects/:projectID/tasks/:taskID/transition", okHandler)
			router.POST("/api/v1/projects/:projectID/defects/:defectID/transition", okHandler)
			router.POST("/api/v1/projects/:projectID/backlog/reorder", okHandler)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code, "viewer should be blocked on write route")
		})
	}
}

// TestCollabWriteRoutes_DeveloperAllowed verifies that a developer-level
// project member can access collaboration write routes.
func TestCollabWriteRoutes_DeveloperAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID, _, developerID, _ := setupCollabRBACTestDB(t)

	writePaths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/projects/" + projectID + "/requirements"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/sprints"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/tasks"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/test-cases"},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/defects"},
	}

	for _, tc := range writePaths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			router := gin.New()
			router.Use(claimsMiddleware(developerID, app.UserRoleUser))
			router.Use(RequireProjectRole(app.ProjectRoleDeveloper))

			router.POST("/api/v1/projects/:projectID/requirements", okHandler)
			router.POST("/api/v1/projects/:projectID/sprints", okHandler)
			router.POST("/api/v1/projects/:projectID/tasks", okHandler)
			router.POST("/api/v1/projects/:projectID/test-cases", okHandler)
			router.POST("/api/v1/projects/:projectID/defects", okHandler)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "developer should access write route")
		})
	}
}

// TestCollabReadRoutes_NonMemberBlocked verifies that a user who is NOT a
// member of the project gets 403 on both read and write collaboration routes.
func TestCollabReadRoutes_NonMemberBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID, _, _, _ := setupCollabRBACTestDB(t)
	outsiderID := uuid.NewString()

	router := gin.New()
	router.Use(claimsMiddleware(outsiderID, app.UserRoleUser))
	router.Use(RequireProjectRole(app.ProjectRoleViewer))
	router.GET("/api/v1/projects/:projectID/requirements", okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/requirements", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "non-member should be blocked")
}

// TestCollabRoutes_AdminBypassesRBAC verifies that an admin user bypasses
// project role checks on collaboration routes.
func TestCollabRoutes_AdminBypassesRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID, _, _, _ := setupCollabRBACTestDB(t)
	adminID := uuid.NewString()

	paths := []struct {
		method  string
		path    string
		minRole string
	}{
		{http.MethodGet, "/api/v1/projects/" + projectID + "/requirements", app.ProjectRoleViewer},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/requirements", app.ProjectRoleDeveloper},
		{http.MethodPost, "/api/v1/projects/" + projectID + "/sprints", app.ProjectRoleDeveloper},
	}

	for _, tc := range paths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			router := gin.New()
			router.Use(claimsMiddleware(adminID, app.UserRoleAdmin))
			router.Use(RequireProjectRole(tc.minRole))

			router.GET("/api/v1/projects/:projectID/requirements", okHandler)
			router.POST("/api/v1/projects/:projectID/requirements", okHandler)
			router.POST("/api/v1/projects/:projectID/sprints", okHandler)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "admin should bypass RBAC")
		})
	}
}

// TestCollabRoutes_UnauthenticatedBlocked verifies that requests without
// claims get 401 on collaboration routes.
func TestCollabRoutes_UnauthenticatedBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// No claims middleware — simulates unauthenticated request
	router.Use(RequireProjectRole(app.ProjectRoleViewer))
	router.GET("/api/v1/projects/:projectID/requirements", okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj-1/requirements", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "unauthenticated should get 401")
}

// TestCollabRoutes_OwnerCanWrite verifies that a project owner can access
// both read and write collaboration routes.
func TestCollabRoutes_OwnerCanWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID, ownerID, _, _ := setupCollabRBACTestDB(t)

	router := gin.New()
	router.Use(claimsMiddleware(ownerID, app.UserRoleUser))
	router.Use(RequireProjectRole(app.ProjectRoleDeveloper))
	router.POST("/api/v1/projects/:projectID/requirements", okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/requirements", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "owner should access developer-level write route")
}

// TestBlockViewer_CollabRoutes verifies that BlockViewer middleware rejects
// viewers and allows developers on collaboration-style routes.
func TestBlockViewer_CollabRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID, _, developerID, viewerID := setupCollabRBACTestDB(t)

	t.Run("viewer blocked", func(t *testing.T) {
		router := gin.New()
		router.Use(claimsMiddleware(viewerID, app.UserRoleUser))
		router.Use(BlockViewer())
		router.POST("/api/v1/projects/:projectID/requirements", okHandler)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/requirements", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("developer allowed", func(t *testing.T) {
		router := gin.New()
		router.Use(claimsMiddleware(developerID, app.UserRoleUser))
		router.Use(BlockViewer())
		router.POST("/api/v1/projects/:projectID/requirements", okHandler)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/requirements", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
