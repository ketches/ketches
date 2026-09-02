package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDeleteApp_IgnoresMissingKubernetesResources(t *testing.T) {
	setupAppVolumeTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
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
		Slug:           "hello-app-1d982e1d",
		Name:           "Hello App",
		EnvID:          "env-1",
		AppType:        "Deployment",
		ContainerImage: "nginx:latest",
		Replicas:       1,
	}).Error)

	require.NoError(t, DeleteApp(context.Background(), "app-1"))

	var deleted entities.App
	err := db.DB.First(&deleted, "id = ?", "app-1").Error
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDeleteAppUsesFenceThroughDatabaseCommit(t *testing.T) {
	setupAppVolumeTestDB(t)
	originalDeleteResources := deleteAppK8sResources
	originalEnsureSharedGateway := ensureSharedGatewayForAppDelete
	t.Cleanup(func() {
		deleteAppK8sResources = originalDeleteResources
		ensureSharedGatewayForAppDelete = originalEnsureSharedGateway
	})

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-fence"},
		Slug: "project-fence",
		Name: "Project Fence",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-fence"},
		Slug:       "cluster-fence",
		Name:       "Cluster Fence",
		KubeConfig: "test",
		Enabled:    true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-fence"},
		Slug:             "env-fence",
		Name:             "Environment Fence",
		ProjectID:        "project-fence",
		ClusterID:        "cluster-fence",
		ClusterNamespace: "work-fence",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-fence"},
		Slug:           "app-fence",
		Name:           "App Fence",
		EnvID:          "env-fence",
		AppType:        "Deployment",
		ContainerImage: "nginx:latest",
		DeployStatus:   "deployed",
	}).Error)

	fenceHeld := make(chan struct{})
	releaseFence := make(chan struct{})
	fenceResult := make(chan error, 1)
	go func() {
		fenceResult <- core.WithAppReconcileFence(context.Background(), "app-fence", func() error {
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

	kubernetesDeleteCalled := make(chan struct{})
	deleteAppK8sResources = func(context.Context, *models.AppContext, bool) error {
		close(kubernetesDeleteCalled)
		return nil
	}
	var reconciledClusterID string
	var observedDeletedCount int64
	var observeErr error
	ensureSharedGatewayForAppDelete = func(_ context.Context, clusterID string) error {
		reconciledClusterID = clusterID
		observeErr = db.DB.Unscoped().Model(&entities.App{}).
			Where("id = ? AND deleted_at IS NOT NULL", "app-fence").
			Count(&observedDeletedCount).Error
		return observeErr
	}

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- DeleteApp(context.Background(), "app-fence")
	}()
	select {
	case <-kubernetesDeleteCalled:
		t.Fatal("app deletion entered Kubernetes while the app fence was held")
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFence)
	require.NoError(t, <-fenceResult)
	require.NoError(t, <-deleteResult)
	require.NoError(t, observeErr)
	require.Equal(t, int64(1), observedDeletedCount)
	require.Equal(t, "cluster-fence", reconciledClusterID)

	var deleted entities.App
	require.NoError(t, db.DB.Unscoped().First(&deleted, "id = ?", "app-fence").Error)
	require.True(t, deleted.DeletedAt.Valid)
	require.Equal(t, "undeployed", deleted.DeployStatus)
}
