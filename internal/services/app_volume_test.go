package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupAppVolumeTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/app-volume-test.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.Project{},
		&entities.Cluster{},
		&entities.Env{},
		&entities.App{},
		&entities.AppVolume{},
		&entities.AppEnvVar{},
		&entities.AppConfigFile{},
		&entities.AppProbe{},
		&entities.AppGateway{},
		&entities.AppAutoScaling{},
		&entities.AppSchedulingRule{},
		&entities.AppPlugin{},
	))

	db.DB = testDB
}

func registerAppVolumeTestCluster(t *testing.T, serverURL string) string {
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

func TestListAppVolumes_LoadsPVCStatusFromCluster(t *testing.T) {
	setupAppVolumeTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/namespaces/work/persistentvolumeclaims":
			require.Equal(t, http.MethodGet, r.Method)
			require.NoError(t, json.NewEncoder(w).Encode(&corev1.PersistentVolumeClaimList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "PersistentVolumeClaimList",
				},
				Items: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "data",
							Namespace: "work",
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase: corev1.ClaimBound,
						},
					},
				},
			}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	clusterID := registerAppVolumeTestCluster(t, server.URL)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo-project",
		Name: "Demo Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: clusterID},
		Slug:       "demo-cluster",
		Name:       "Demo Cluster",
		KubeConfig: "test",
		Enabled:    true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "prod",
		Name:             "Production",
		ProjectID:        "project-1",
		ClusterID:        clusterID,
		ClusterNamespace: "work",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-1"},
		Slug:           "api",
		Name:           "API",
		EnvID:          "env-1",
		AppType:        "Deployment",
		ContainerImage: "nginx:latest",
		Replicas:       1,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppVolume{
		ID:         "vol-1",
		AppID:      "app-1",
		Slug:       "data",
		MountPath:  "/data",
		VolumeType: "pvc",
		Capacity:   10,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppVolume{
		ID:         "vol-2",
		AppID:      "app-1",
		Slug:       "cache",
		MountPath:  "/cache",
		VolumeType: "emptyDir",
		Capacity:   1,
	}).Error)

	volumes, err := ListAppVolumes("app-1")
	require.NoError(t, err)
	require.Len(t, volumes, 2)

	statusBySlug := make(map[string]string, len(volumes))
	for _, volume := range volumes {
		statusBySlug[volume.Slug] = volume.Status
	}

	assert.Equal(t, "Bound", statusBySlug["data"])
	assert.Empty(t, statusBySlug["cache"])
}
