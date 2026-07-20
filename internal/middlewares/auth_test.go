package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupAuthMiddlewareTestDB(t *testing.T) {
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
	require.NoError(t, testDB.AutoMigrate(&entities.User{}))

	db.DB = testDB
	app.Config.JWTSecret = "middleware-auth-test-secret"
}

func seedAuthMiddlewareUser(t *testing.T) *entities.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &entities.User{
		Base:     entities.Base{ID: "user-1"},
		Username: "alice",
		Email:    "alice@example.com",
		Password: string(hashedPassword),
		Role:     app.UserRoleUser,
	}
	require.NoError(t, db.DB.Create(user).Error)

	return user
}

func TestAuthAcceptsAccessTokenCookie(t *testing.T) {
	setupAuthMiddlewareTestDB(t)
	user := seedAuthMiddlewareUser(t)

	token, err := app.GenerateAccessToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth())
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: app.AccessTokenCookieName, Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthRejectsQueryTokenParameter(t *testing.T) {
	setupAuthMiddlewareTestDB(t)
	user := seedAuthMiddlewareUser(t)

	token, err := app.GenerateAccessToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth())
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected?token="+token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePasswordChangeBlocksBusinessRoutes(t *testing.T) {
	setupAuthMiddlewareTestDB(t)
	user := seedAuthMiddlewareUser(t)
	require.NoError(t, db.DB.Model(user).Update("must_change_password", true).Error)
	user.MustChangePassword = true

	token, err := app.GenerateAccessToken(user)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(), RequirePasswordChange())
	r.GET("/api/v1/projects", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/v1/users/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.PATCH("/api/v1/users/me/password", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.POST("/api/v1/users/logout", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	tests := []struct {
		method string
		path   string
		status int
	}{
		{method: http.MethodGet, path: "/api/v1/projects", status: http.StatusForbidden},
		{method: http.MethodGet, path: "/api/v1/users/me", status: http.StatusNoContent},
		{method: http.MethodPatch, path: "/api/v1/users/me/password", status: http.StatusNoContent},
		{method: http.MethodPost, path: "/api/v1/users/logout", status: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code)
		})
	}
}
