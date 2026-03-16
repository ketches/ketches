package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupOperationLogHandlerTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.OperationLog{}, &entities.SystemSetting{}))
	db.DB = testDB
}

func TestListOperationLogsExportCSV(t *testing.T) {
	setupOperationLogHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, db.DB.Create(&entities.OperationLog{Base: entities.Base{ID: "l1"}, Username: "alice", Action: "deploy", ResourceType: "app", Status: entities.OperationLogStatusSuccess, StatusCode: 200, Sensitivity: entities.OperationLogSensitivityPublic}).Error)

	r := gin.New()
	r.GET("/api/v1/operation-logs", ListOperationLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operation-logs?export=true&page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment;")
	assert.Contains(t, w.Body.String(), "id,created_at")
}

func TestGetAndUpdateOperationLogSettings(t *testing.T) {
	setupOperationLogHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/operation-logs/settings", GetOperationLogSettings)
	r.PUT("/api/v1/operation-logs/settings", UpdateOperationLogSettings)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/operation-logs/settings", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var getResp map[string]map[string]any
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &getResp))
	assert.Equal(t, float64(90), getResp["data"]["retention_days"])

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/operation-logs/settings", strings.NewReader(`{"retention_days":120}`))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	assert.Equal(t, http.StatusNoContent, putW.Code)

	getReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/operation-logs/settings", nil)
	getW2 := httptest.NewRecorder()
	r.ServeHTTP(getW2, getReq2)
	assert.Equal(t, http.StatusOK, getW2.Code)

	var getResp2 map[string]map[string]any
	require.NoError(t, json.Unmarshal(getW2.Body.Bytes(), &getResp2))
	assert.Equal(t, float64(120), getResp2["data"]["retention_days"])
}

func TestListPlatformAuditLogsFiltersPlatformOperations(t *testing.T) {
	setupOperationLogHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	require.NoError(t, db.DB.Create(&entities.OperationLog{
		Base:         entities.Base{ID: "platform-log"},
		Username:     "alice",
		Action:       "update",
		ResourceType: "platform_branding",
		Status:       entities.OperationLogStatusSuccess,
		StatusCode:   200,
		Sensitivity:  entities.OperationLogSensitivitySensitive,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.OperationLog{
		Base:         entities.Base{ID: "app-log"},
		Username:     "bob",
		Action:       "deploy",
		ResourceType: "app",
		Status:       entities.OperationLogStatusSuccess,
		StatusCode:   200,
		Sensitivity:  entities.OperationLogSensitivityInternal,
	}).Error)

	r := gin.New()
	r.GET("/api/v1/platform-settings/audit-logs", ListPlatformAuditLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform-settings/audit-logs?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "platform_branding", resp.Data.Items[0]["resource_type"])
}
