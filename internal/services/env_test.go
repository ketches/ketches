package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupEnvTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/env-test.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.Project{},
		&entities.Cluster{},
		&entities.ClusterIntegration{},
		&entities.Env{},
		&entities.EnvResourceQuota{},
		&entities.Certificate{},
	))

	db.DB = testDB
}

func registerEnvTestCluster(t *testing.T, serverURL string) string {
	t.Helper()

	clusterID := "cluster-" + t.Name()
	kubeConfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
users:
- name: test
  user: {}
`, serverURL)

	require.NoError(t, kube.GlobalClusterStore.AddClient(clusterID, kubeConfig))
	t.Cleanup(func() {
		kube.GlobalClusterStore.RemoveClient(clusterID)
	})

	return clusterID
}

func seedEnvTestProjectAndCluster(t *testing.T, clusterID string) {
	t.Helper()

	project := entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo-project",
		Name: "Demo Project",
	}
	cluster := entities.Cluster{
		Base:       entities.Base{ID: clusterID},
		Slug:       "demo-cluster",
		Name:       "Demo Cluster",
		KubeConfig: "test",
		Enabled:    true,
	}

	require.NoError(t, db.DB.Create(&project).Error)
	require.NoError(t, db.DB.Create(&cluster).Error)
}

func newKubeNotFoundStatus(name string) map[string]any {
	return map[string]any{
		"kind":       "Status",
		"apiVersion": "v1",
		"status":     "Failure",
		"message":    fmt.Sprintf("namespaces %q not found", name),
		"reason":     "NotFound",
		"code":       http.StatusNotFound,
	}
}

func TestCreateEnv_UsesProvidedNamespaceWithoutRandomSuffix(t *testing.T) {
	setupEnvTestDB(t)

	var createdNamespace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/namespaces/demo-project-production":
			w.WriteHeader(http.StatusNotFound)
			require.NoError(t, json.NewEncoder(w).Encode(newKubeNotFoundStatus("demo-project-production")))
		case "/api/v1/namespaces":
			require.Equal(t, http.MethodPost, r.Method)

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), "demo-project-production")

			createdNamespace = "demo-project-production"
			w.WriteHeader(http.StatusCreated)
			require.NoError(t, json.NewEncoder(w).Encode(&corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{Name: createdNamespace},
			}))
		case "/apis/gateway.networking.k8s.io/v1":
			w.WriteHeader(http.StatusNotFound)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"kind":       "Status",
				"apiVersion": "v1",
				"status":     "Failure",
				"reason":     "NotFound",
				"code":       http.StatusNotFound,
			}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	clusterID := registerEnvTestCluster(t, server.URL)
	seedEnvTestProjectAndCluster(t, clusterID)

	env, err := CreateEnv("project-1", &models.CreateEnvRequest{
		Name:             "Production",
		Slug:             "production",
		ClusterID:        clusterID,
		ClusterNamespace: "demo-project-production",
	})
	require.NoError(t, err)

	assert.Equal(t, "demo-project-production", env.ClusterNamespace)
	assert.Equal(t, "demo-project-production", createdNamespace)

	var stored entities.Env
	require.NoError(t, db.DB.WithContext(context.Background()).First(&stored, "id = ?", env.ID).Error)
	assert.Equal(t, "demo-project-production", stored.ClusterNamespace)
}

func TestCreateEnv_RejectsExistingClusterNamespace(t *testing.T) {
	setupEnvTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/namespaces/demo-project-production":
			require.Equal(t, http.MethodGet, r.Method)
			require.NoError(t, json.NewEncoder(w).Encode(&corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{Name: "demo-project-production"},
			}))
		case "/api/v1/namespaces":
			t.Fatalf("unexpected namespace create request for existing namespace")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	clusterID := registerEnvTestCluster(t, server.URL)
	seedEnvTestProjectAndCluster(t, clusterID)

	_, err := CreateEnv("project-1", &models.CreateEnvRequest{
		Name:             "Production",
		Slug:             "production",
		ClusterID:        clusterID,
		ClusterNamespace: "demo-project-production",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace")
	assert.Contains(t, err.Error(), "already exists")

	var count int64
	require.NoError(t, db.DB.Model(&entities.Env{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCheckEnvNamespaceAvailability_ReturnsDatabaseConflict(t *testing.T) {
	setupEnvTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "production",
		Name:             "Production",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "shared-prod",
	}).Error)

	res, err := CheckEnvNamespaceAvailability("cluster-1", "shared-prod")
	require.NoError(t, err)

	assert.False(t, res.Available)
	assert.Equal(t, "database", res.Source)
	assert.Contains(t, res.Message, "already used by another environment")
}

func TestCheckEnvNamespaceAvailability_ReturnsClusterConflict(t *testing.T) {
	setupEnvTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/namespaces/shared-prod":
			require.Equal(t, http.MethodGet, r.Method)
			require.NoError(t, json.NewEncoder(w).Encode(&corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{Name: "shared-prod"},
			}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	clusterID := registerEnvTestCluster(t, server.URL)
	seedEnvTestProjectAndCluster(t, clusterID)

	res, err := CheckEnvNamespaceAvailability(clusterID, "shared-prod")
	require.NoError(t, err)

	assert.False(t, res.Available)
	assert.Equal(t, "cluster", res.Source)
	assert.Contains(t, res.Message, "already exists in the selected cluster")
}

func TestGetEnvWithProjectNameIncludesClusterStatus(t *testing.T) {
	setupEnvTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:                   entities.Base{ID: "cluster-1"},
		Slug:                   "cluster-1",
		Name:                   "Cluster 1",
		KubeConfig:             "test",
		Enabled:                true,
		ConnectionStatus:       "connected",
		ConnectionStatusReason: "healthy",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "env-1",
		Name:             "Env 1",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "ns-1",
	}).Error)

	resp, err := GetEnvWithProjectName("env-1")
	require.NoError(t, err)
	assert.Equal(t, "Project 1", resp.ProjectName)
	assert.Equal(t, "Cluster 1", resp.ClusterName)
	assert.Equal(t, "connected", resp.ClusterConnectionStatus)
	assert.Equal(t, "healthy", resp.ClusterConnectionStatusReason)
}
