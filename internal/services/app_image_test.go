package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAppImageTags_FiltersAndReversesTags(t *testing.T) {
	setupAppVolumeTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case "/v2/demo/tags/list":
			require.Equal(t, http.MethodGet, r.Method)
			_, err := w.Write([]byte(`{"tags":["v1.2.0","buildcache-123","v1.10.0","v1.3.0"]}`))
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	registryURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	repository := fmt.Sprintf("%s/demo", registryURL.Host)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo-project",
		Name: "Demo Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
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
		ClusterID:        "cluster-1",
		ClusterNamespace: "work",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-1"},
		Slug:           "api",
		Name:           "API",
		EnvID:          "env-1",
		AppType:        "Deployment",
		ContainerImage: repository + ":v1.2.0",
		Replicas:       1,
	}).Error)

	result, err := ListAppImageTags(context.Background(), "app-1")
	require.NoError(t, err)

	assert.Equal(t, repository, result.Repository)
	assert.Equal(t, "v1.2.0", result.CurrentTag)
	assert.Equal(t, []string{"v1.3.0", "v1.10.0", "v1.2.0"}, result.Tags)
}
