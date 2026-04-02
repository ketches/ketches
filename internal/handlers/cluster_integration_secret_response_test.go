package handlers

import (
	"encoding/json"
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
	"gorm.io/gorm"
)

func setupClusterIntegrationHandlerSecretTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.ClusterIntegration{}))

	db.DB = testDB
}

func TestGetClusterIntegrationDoesNotExposeSecrets(t *testing.T) {
	setupClusterIntegrationHandlerSecretTestDB(t)

	require.NoError(t, db.DB.Create(&entities.ClusterIntegration{
		ID:              "integration-1",
		ClusterID:       "cluster-1",
		IntegrationType: entities.IntegrationTypePrometheus,
		Name:            "Prometheus",
		Endpoint:        "https://prometheus.example.com",
		Password:        "enc:v1:opaque",
		Token:           "enc:v1:opaque",
		CACert:          "enc:v1:opaque",
		Enabled:         true,
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/clusters/:clusterID/integrations/:integrationID", GetClusterIntegration)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/cluster-1/integrations/integration-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	_, passwordExists := data["password"]
	_, tokenExists := data["token"]
	_, caCertExists := data["ca_cert"]
	assert.False(t, passwordExists)
	assert.False(t, tokenExists)
	assert.False(t, caCertExists)
}

func TestListClusterIntegrationsDoesNotExposeSecrets(t *testing.T) {
	setupClusterIntegrationHandlerSecretTestDB(t)

	require.NoError(t, db.DB.Create(&entities.ClusterIntegration{
		ID:              "integration-1",
		ClusterID:       "cluster-1",
		IntegrationType: entities.IntegrationTypePrometheus,
		Name:            "Prometheus",
		Endpoint:        "https://prometheus.example.com",
		Password:        "enc:v1:opaque",
		Token:           "enc:v1:opaque",
		CACert:          "enc:v1:opaque",
		Enabled:         true,
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(accessControlClaimsMiddleware("admin-1", "admin", app.UserRoleAdmin))
	r.GET("/api/v1/clusters/:clusterID/integrations", ListClusterIntegrations)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/cluster-1/integrations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	require.Len(t, data, 1)
	item := data[0].(map[string]any)
	_, passwordExists := item["password"]
	_, tokenExists := item["token"]
	_, caCertExists := item["ca_cert"]
	assert.False(t, passwordExists)
	assert.False(t, tokenExists)
	assert.False(t, caCertExists)
}
