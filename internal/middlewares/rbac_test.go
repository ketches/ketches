package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/stretchr/testify/assert"
)

// TestRequireProjectRole_AdminBypasses verifies that a user with the "admin"
// system role is allowed through RequireProjectRole without any DB lookup.
func TestRequireProjectRole_AdminBypasses(t *testing.T) {
	router := gin.New()

	// Inject admin claims into context (no real JWT needed)
	router.Use(func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID: "admin-user-id",
			Role:   "admin",
		})
		c.Next()
	})

	router.Use(RequireProjectRole("owner"))
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
			Role:   "admin",
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
	router.Use(RequireProjectRole("viewer"))
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
			Role:   "user",
		})
		c.Next()
	})

	router.Use(RequireProjectRole("viewer"))
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
			Role:   "user",
		})
		c.Next()
	})

	router.Use(RequireProjectRole("viewer"))
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
	assert.Equal(t, 3, projectRoleRank["owner"])
	assert.Equal(t, 2, projectRoleRank["developer"])
	assert.Equal(t, 1, projectRoleRank["viewer"])
	assert.Equal(t, 0, projectRoleRank["nonexistent"])
}
