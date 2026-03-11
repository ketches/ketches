package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOperationLogMiddlewareTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.OperationLog{}))
	db.DB = testDB
}

func TestOperationLogMiddlewareBodyRestoreAndMappedRoute(t *testing.T) {
	setupOperationLogMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OperationLog())
	r.POST("/api/v1/apps/:appID/action", func(c *gin.Context) {
		var req map[string]any
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"action": req["action"]})
	})

	body, _ := json.Marshal(map[string]any{"action": "deploy"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/apps/a1/action", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var logs []entities.OperationLog
	require.NoError(t, db.DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, "deploy", logs[0].Action)
	assert.Equal(t, "app", logs[0].ResourceType)
	assert.Equal(t, "a1", logs[0].ResourceID)
}

func TestOperationLogMiddlewareSkipsUnmappedRoute(t *testing.T) {
	setupOperationLogMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OperationLog())
	r.POST("/api/v1/custom/unmapped", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/custom/unmapped", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var count int64
	require.NoError(t, db.DB.Model(&entities.OperationLog{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestOperationLogMiddlewareSignInLogsWithBodyUsername(t *testing.T) {
	setupOperationLogMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OperationLog())
	r.POST("/api/v1/users/sign-in", func(c *gin.Context) {
		var req map[string]any
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body, _ := json.Marshal(map[string]any{"username": "tester", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/sign-in", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var logs []entities.OperationLog
	require.NoError(t, db.DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, "sign_in", logs[0].Action)
	assert.Equal(t, "tester", logs[0].Username)
}
