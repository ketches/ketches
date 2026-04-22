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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupAuthSessionHandlerTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	originalConfig := app.Config
	t.Cleanup(func() {
		db.DB = originalDB
		app.Config = originalConfig
	})

	testDB, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.User{}))

	db.DB = testDB
	app.Config.JWTSecret = "auth-session-handler-test-secret"
	app.Config.JWTIssuer = "ketches.test"
	app.Config.JWTAudience = "ketches-ui"
	app.Config.AccessTokenTTLMinutes = 60
	app.Config.RefreshTokenTTLHours = 24 * 7
}

func seedAuthSessionHandlerUser(t *testing.T, username, password string) *entities.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &entities.User{
		Base:     entities.Base{ID: "user-1"},
		Username: username,
		Email:    username + "@example.com",
		Password: string(hashedPassword),
		Role:     app.UserRoleUser,
	}
	require.NoError(t, db.DB.Create(user).Error)
	return user
}

func TestSignInSetsSessionCookiesAndReturnsUserOnly(t *testing.T) {
	setupAuthSessionHandlerTestDB(t)
	seedAuthSessionHandlerUser(t, "alice", "Password#123")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/users/sign-in", SignIn)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/sign-in", strings.NewReader(`{"username":"alice","password":"Password#123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	setCookieHeaders := w.Result().Header.Values("Set-Cookie")
	assert.NotEmpty(t, setCookieHeaders)
	assert.Contains(t, strings.Join(setCookieHeaders, "\n"), app.AccessTokenCookieName+"=")
	assert.Contains(t, strings.Join(setCookieHeaders, "\n"), app.RefreshTokenCookieName+"=")
	assert.Contains(t, strings.Join(setCookieHeaders, "\n"), app.CSRFCookieName+"=")
	assert.NotContains(t, w.Body.String(), "access_token")
	assert.NotContains(t, w.Body.String(), "refresh_token")

	var resp struct {
		Data struct {
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "alice", resp.Data.User.Username)
}

func TestRefreshTokenRotatesSessionCookies(t *testing.T) {
	setupAuthSessionHandlerTestDB(t)
	seedAuthSessionHandlerUser(t, "alice", "Password#123")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/users/sign-in", SignIn)
	r.POST("/api/v1/users/refresh-token", middlewares.CSRF(), RefreshToken)

	signInReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/sign-in", strings.NewReader(`{"username":"alice","password":"Password#123"}`))
	signInReq.Header.Set("Content-Type", "application/json")
	signInRes := httptest.NewRecorder()
	r.ServeHTTP(signInRes, signInReq)
	require.Equal(t, http.StatusOK, signInRes.Code)

	cookies := signInRes.Result().Cookies()
	require.NotEmpty(t, cookies)

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/refresh-token", nil)
	var csrfToken string
	for _, cookie := range cookies {
		refreshReq.AddCookie(cookie)
		if cookie.Name == app.CSRFCookieName {
			csrfToken = cookie.Value
		}
	}
	require.NotEmpty(t, csrfToken)
	refreshReq.Header.Set(app.CSRFHeaderName, csrfToken)

	refreshRes := httptest.NewRecorder()
	r.ServeHTTP(refreshRes, refreshReq)

	require.Equal(t, http.StatusOK, refreshRes.Code)
	assert.Contains(t, strings.Join(refreshRes.Result().Header.Values("Set-Cookie"), "\n"), app.AccessTokenCookieName+"=")
	assert.Contains(t, strings.Join(refreshRes.Result().Header.Values("Set-Cookie"), "\n"), app.RefreshTokenCookieName+"=")
}
