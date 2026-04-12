package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
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
