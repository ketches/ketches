package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type userAPIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func setupUserHandlerTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(&entities.User{}))

	db.DB = testDB
}

func userClaimsMiddleware(userID, username, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID:   userID,
			Username: username,
			Role:     role,
		})
		c.Next()
	}
}

func seedUserHandlerTestUser(t *testing.T, userID, username, password, role string) {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	require.NoError(t, db.DB.Create(&entities.User{
		Base:     entities.Base{ID: userID},
		Username: username,
		Email:    username + "@example.com",
		Password: string(hashedPassword),
		Fullname: "Original Name",
		Bio:      "Original bio",
		Role:     role,
	}).Error)
}

func TestGetCurrentUserProfile(t *testing.T) {
	setupUserHandlerTestDB(t)
	seedUserHandlerTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/users")
	group.Use(userClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	group.GET("/me", GetCurrentUserProfile)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp userAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Empty(t, resp.Error)

	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, "Original Name", data["fullname"])
	assert.Equal(t, "Original bio", data["bio"])
}

func TestUpdateCurrentUserProfile(t *testing.T) {
	setupUserHandlerTestDB(t)
	seedUserHandlerTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/users")
	group.Use(userClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	group.PUT("/me/profile", UpdateCurrentUserProfile)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/profile", strings.NewReader(`{"fullname":"Alice Example","email":"alice+new@example.com","bio":"Updated bio"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var persisted entities.User
	require.NoError(t, db.DB.First(&persisted, "id = ?", "user-1").Error)
	assert.Equal(t, "Alice Example", persisted.Fullname)
	assert.Equal(t, "alice+new@example.com", persisted.Email)
	assert.Equal(t, "Updated bio", persisted.Bio)
}

func TestChangeCurrentUserPassword(t *testing.T) {
	setupUserHandlerTestDB(t)
	seedUserHandlerTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/users")
	group.Use(userClaimsMiddleware("user-1", "alice", app.UserRoleUser))
	group.PATCH("/me/password", ChangeCurrentUserPassword)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me/password", strings.NewReader(`{"current_password":"secret123","new_password":"new-secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)

	var persisted entities.User
	require.NoError(t, db.DB.First(&persisted, "id = ?", "user-1").Error)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(persisted.Password), []byte("new-secret123")))
}

func TestAdminChangeUserPassword(t *testing.T) {
	setupUserHandlerTestDB(t)
	seedUserHandlerTestUser(t, "user-1", "alice", "secret123", app.UserRoleUser)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/users")
	group.Use(userClaimsMiddleware("admin-1", "admin", app.UserRoleAdmin))
	group.PATCH("/:userID/password", middlewares.AdminOnly(), ChangeUserPassword)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/user-1/password", strings.NewReader(`{"password":"admin-reset123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)

	var persisted entities.User
	require.NoError(t, db.DB.First(&persisted, "id = ?", "user-1").Error)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(persisted.Password), []byte("admin-reset123")))
}
