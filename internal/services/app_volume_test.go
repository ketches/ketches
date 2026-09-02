package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
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

func setupAppVolumeTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	originalValidateDelete := validateAppVolumeDeleteForService
	originalDeleteFromK8s := deleteVolumeFromK8sForService
	t.Cleanup(func() {
		db.DB = originalDB
		validateAppVolumeDeleteForService = originalValidateDelete
		deleteVolumeFromK8sForService = originalDeleteFromK8s
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

func TestValidateAppVolumeRequestDisablesHostPathByDefault(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.AllowHostPathVolumes = false

	err := validateAppVolumeRequest(app.VolumeTypeHostPath, "/var/lib/ketches")
	require.ErrorContains(t, err, "disabled")
	require.NoError(t, validateAppVolumeRequest(app.VolumeTypeEmptyDir, ""))
}

func TestValidateAppVolumeRequestAllowsSafeHostPathWhenEnabled(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.AllowHostPathVolumes = true

	require.NoError(t, validateAppVolumeRequest(app.VolumeTypeHostPath, "/var/lib/ketches"))
	require.Error(t, validateAppVolumeRequest(app.VolumeTypeHostPath, "/"))
	require.Error(t, validateAppVolumeRequest(app.VolumeTypeHostPath, "relative/path"))
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

func TestDeleteAppVolumeUsesAppReconcileFence(t *testing.T) {
	setupAppVolumeTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-delete"},
		Slug: "project-delete",
		Name: "Project Delete",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-delete"},
		Slug:       "cluster-delete",
		Name:       "Cluster Delete",
		KubeConfig: "test",
		Enabled:    true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-delete"},
		Slug:             "env-delete",
		Name:             "Environment Delete",
		ProjectID:        "project-delete",
		ClusterID:        "cluster-delete",
		ClusterNamespace: "work-delete",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-delete-volume"},
		Slug:           "app-delete-volume",
		Name:           "App Delete Volume",
		EnvID:          "env-delete",
		AppType:        app.AppTypeDeployment,
		ContainerImage: "nginx:latest",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppVolume{
		ID:         "volume-delete",
		AppID:      "app-delete-volume",
		Slug:       "data",
		MountPath:  "/data",
		VolumeType: app.VolumeTypePVC,
		Capacity:   10,
	}).Error)

	fenceHeld := make(chan struct{})
	releaseFence := make(chan struct{})
	fenceResult := make(chan error, 1)
	go func() {
		fenceResult <- core.WithAppReconcileFence(context.Background(), "app-delete-volume", func() error {
			close(fenceHeld)
			<-releaseFence
			return nil
		})
	}()
	select {
	case <-fenceHeld:
	case <-time.After(5 * time.Second):
		t.Fatal("test fence was not acquired")
	}

	validateAppVolumeDeleteForService = func(context.Context, *models.AppContext, *entities.AppVolume) error {
		return nil
	}
	kubernetesDeleteCalled := make(chan struct{})
	deleteVolumeFromK8sForService = func(context.Context, *models.AppContext, *entities.AppVolume) error {
		close(kubernetesDeleteCalled)
		return nil
	}
	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- DeleteAppVolume(context.Background(), "volume-delete")
	}()

	select {
	case <-kubernetesDeleteCalled:
		t.Fatal("volume deletion entered Kubernetes while the app fence was held")
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseFence)
	require.NoError(t, <-fenceResult)
	require.NoError(t, <-deleteResult)

	var count int64
	require.NoError(t, db.DB.Model(&entities.AppVolume{}).Where("id = ?", "volume-delete").Count(&count).Error)
	require.Zero(t, count)
}
