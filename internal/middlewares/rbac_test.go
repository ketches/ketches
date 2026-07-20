package middlewares

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRBACTestDB(t *testing.T) {
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
		&entities.Env{},
		&entities.App{},
	))

	db.DB = testDB
}

// TestRequireProjectRole_AdminBypasses verifies that a user with the "admin"
// system role is allowed through RequireProjectRole without any DB lookup.
func TestRequireProjectRole_AdminBypasses(t *testing.T) {
	router := gin.New()

	// Inject admin claims into context (no real JWT needed)
	router.Use(func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID: "admin-user-id",
			Role:   app.UserRoleAdmin,
		})
		c.Next()
	})

	router.Use(RequireProjectRole(app.ProjectRoleOwner))
	router.GET("/test/:projectID", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test/proj-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

// TestBlockViewer_AdminBypasses verifies that a user with the "admin" system
// role is allowed through BlockViewer without any DB lookup.
func TestBlockViewer_AdminBypasses(t *testing.T) {
	router := gin.New()

	// Inject admin claims into context
	router.Use(func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID: "admin-user-id",
			Role:   app.UserRoleAdmin,
		})
		c.Next()
	})

	router.Use(BlockViewer())
	router.GET("/test/:projectID", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test/proj-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

// TestRequireProjectRole_NoClaims verifies that a request without claims in
// the context results in a 401 Unauthorized response.
func TestRequireProjectRole_NoClaims(t *testing.T) {
	router := gin.New()
	router.Use(RequireProjectRole(app.ProjectRoleViewer))
	router.GET("/test/:projectID", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test/proj-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestRequireProjectRole_NoProjectID verifies that a request without any
// resolvable project ID results in a 400 Bad Request response.
func TestRequireProjectRole_NoProjectID(t *testing.T) {
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID: "user-123",
			Role:   app.UserRoleUser,
		})
		c.Next()
	})

	router.Use(RequireProjectRole(app.ProjectRoleViewer))
	// Route has no :projectID, :envID, or :appID params
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRequireProjectRole_NonMemberGetsForbidden verifies that a non-member
// user gets 403 when the DB is available, or the test is skipped if DB is nil.
func TestRequireProjectRole_NonMemberGetsForbidden(t *testing.T) {
	if db.DB == nil {
		t.Skip("DB not initialized")
	}

	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID: "non-member-user",
			Role:   app.UserRoleUser,
		})
		c.Next()
	})

	router.Use(RequireProjectRole(app.ProjectRoleViewer))
	router.GET("/test/:projectID", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test/proj-no-access", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestProjectRoleRank verifies the role ranking map has correct values.
func TestProjectRoleRank(t *testing.T) {
	assert.Equal(t, 3, projectRoleRank[app.ProjectRoleOwner])
	assert.Equal(t, 2, projectRoleRank[app.ProjectRoleDeveloper])
	assert.Equal(t, 1, projectRoleRank[app.ProjectRoleViewer])
	assert.Equal(t, 0, projectRoleRank["nonexistent"])
}

func TestRequireProjectRole_BatchDeleteResolvesProjectFromBody(t *testing.T) {
	setupRBACTestDB(t)
	gin.SetMode(gin.TestMode)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          "member-1",
		ProjectID:   "project-1",
		UserID:      "user-123",
		ProjectRole: app.ProjectRoleDeveloper,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-1"},
		Slug:      "env-1",
		Name:      "Env 1",
		ProjectID: "project-1",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-1"},
		Slug:           "app-1",
		Name:           "App 1",
		EnvID:          "env-1",
		ContainerImage: "nginx:latest",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-2"},
		Slug:           "app-2",
		Name:           "App 2",
		EnvID:          "env-1",
		ContainerImage: "nginx:latest",
	}).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID: "user-123",
			Role:   app.UserRoleUser,
		})
		c.Next()
	})
	router.Use(RequireProjectRole(app.ProjectRoleDeveloper))
	router.POST("/api/v1/apps/batch-delete", func(c *gin.Context) {
		var req struct {
			IDs []string `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": len(req.IDs)})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/apps/batch-delete", strings.NewReader(`{"ids":["app-1","app-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Count int `json:"count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Count)
}

func TestRequireProjectRole_BatchDeleteRejectsMixedProjects(t *testing.T) {
	setupRBACTestDB(t)
	gin.SetMode(gin.TestMode)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-2"},
		Slug: "project-2",
		Name: "Project 2",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          "member-1",
		ProjectID:   "project-1",
		UserID:      "user-123",
		ProjectRole: app.ProjectRoleDeveloper,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-1"},
		Slug:      "env-1",
		Name:      "Env 1",
		ProjectID: "project-1",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-2"},
		Slug:      "env-2",
		Name:      "Env 2",
		ProjectID: "project-2",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-1"},
		Slug:           "app-1",
		Name:           "App 1",
		EnvID:          "env-1",
		ContainerImage: "nginx:latest",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-2"},
		Slug:           "app-2",
		Name:           "App 2",
		EnvID:          "env-2",
		ContainerImage: "nginx:latest",
	}).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID: "user-123",
			Role:   app.UserRoleUser,
		})
		c.Next()
	})
	router.Use(RequireProjectRole(app.ProjectRoleDeveloper))
	router.POST("/api/v1/apps/batch-delete", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/apps/batch-delete", strings.NewReader(`{"ids":["app-1","app-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRequireRecycleBinOwnerChecksEveryProject(t *testing.T) {
	setupRBACTestDB(t)
	gin.SetMode(gin.TestMode)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-recycle"},
		Slug: "project-recycle",
		Name: "Recycle Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          "member-recycle-owner",
		ProjectID:   "project-recycle",
		UserID:      "owner-1",
		ProjectRole: app.ProjectRoleOwner,
	}).Error)

	newRouter := func(userID string, role string) *gin.Engine {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("claims", &app.Claims{UserID: userID, Role: role})
			c.Next()
		})
		router.POST("/recycle-bin/projects/permanently-delete", RequireRecycleBinOwner("projects"), func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})
		return router
	}

	request := func(router *gin.Engine) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/recycle-bin/projects/permanently-delete", strings.NewReader(`{"ids":["project-recycle"]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	assert.Equal(t, http.StatusOK, request(newRouter("owner-1", app.UserRoleUser)).Code)
	assert.Equal(t, http.StatusForbidden, request(newRouter("viewer-1", app.UserRoleUser)).Code)
	assert.Equal(t, http.StatusOK, request(newRouter("admin-1", app.UserRoleAdmin)).Code)
}
