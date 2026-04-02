package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupClusterHandlerSecretTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Cluster{}, &entities.ClusterIntegration{}))

	db.DB = testDB
}

func TestGetClusterDoesNotExposeKubeConfig(t *testing.T) {
	setupClusterHandlerSecretTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:        entities.Base{ID: "cluster-1"},
		Slug:        "demo-cluster",
		Name:        "Demo Cluster",
		KubeConfig:  "enc:v1:opaque",
		ApiServer:   "https://10.0.0.1:6443",
		GatewayHost: "gateway.example.com",
		Enabled:     true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterIntegration{
		ID:              "integration-1",
		ClusterID:       "cluster-1",
		IntegrationType: entities.IntegrationTypePrometheus,
		Name:            "Prometheus",
		Endpoint:        "https://prometheus.example.com",
		Enabled:         true,
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/clusters/:clusterID", GetCluster)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/cluster-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	_, exists := data["kube_config"]
	assert.False(t, exists)
	assert.Equal(t, "gateway.example.com", data["gateway_host"])
	assert.Equal(t, "https://10.0.0.1:6443", data["api_server"])
	assert.Equal(t, true, data["has_kube_config"])
	assert.Equal(t, true, data["has_prometheus_integration"])
}

func TestListClustersDoesNotExposeKubeConfig(t *testing.T) {
	setupClusterHandlerSecretTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:        entities.Base{ID: "cluster-1"},
		Slug:        "demo-cluster",
		Name:        "Demo Cluster",
		KubeConfig:  "enc:v1:opaque",
		ApiServer:   "https://10.0.0.1:6443",
		GatewayHost: "gateway.example.com",
		Enabled:     true,
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/clusters", ListClusters)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	_, exists := item["kube_config"]
	assert.False(t, exists)
	assert.Equal(t, "gateway.example.com", item["gateway_host"])
	assert.Equal(t, "https://10.0.0.1:6443", item["api_server"])
	assert.Equal(t, true, item["has_kube_config"])
}
