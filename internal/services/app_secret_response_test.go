package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToAppResponseOmitsRegistryPassword(t *testing.T) {
	resp := ToAppResponse(context.Background(), &models.AppContext{
		App: entities.App{
			Base:             entities.Base{ID: "app-1"},
			Slug:             "demo-app",
			Name:             "Demo App",
			EnvID:            "env-1",
			AppType:          app.AppTypeDeployment,
			ContainerImage:   "ghcr.io/demo/app:latest",
			RegistryUsername: "demo",
			RegistryPassword: "enc:v1:opaque",
			Replicas:         1,
		},
		EnvContext: models.EnvContext{
			Env: entities.Env{
				Base:             entities.Base{ID: "env-1"},
				Name:             "Production",
				ProjectID:        "project-1",
				ClusterID:        "cluster-1",
				ClusterNamespace: "prod",
			},
			Project: entities.Project{
				Base: entities.Base{ID: "project-1"},
				Name: "Demo Project",
			},
			Cluster: entities.Cluster{
				Base:             entities.Base{ID: "cluster-1"},
				Name:             "Demo Cluster",
				ConnectionStatus: "connected",
			},
		},
	})

	assert.Empty(t, resp.RegistryPassword)
	assert.True(t, resp.HasRegistryPassword)
	require.NotNil(t, resp.Env)
	assert.Equal(t, "project-1", resp.Env.ProjectID)
	assert.Equal(t, "Demo Project", resp.Env.ProjectName)
	assert.Equal(t, "cluster-1", resp.Env.ClusterID)
	assert.Equal(t, "Demo Cluster", resp.Env.ClusterName)
	assert.Equal(t, "connected", resp.Env.ClusterConnectionStatus)
	assert.False(t, resp.Env.HasPrometheusIntegration)
}

func TestToAppListResponseOmitsRegistryPassword(t *testing.T) {
	resp := ToAppListResponse(context.Background(), &models.AppListRow{
		ID:               "app-1",
		Slug:             "demo-app",
		Name:             "Demo App",
		EnvID:            "env-1",
		AppType:          app.AppTypeDeployment,
		ContainerImage:   "ghcr.io/demo/app:latest",
		RegistryUsername: "demo",
		RegistryPassword: "enc:v1:opaque",
		Replicas:         1,
	})

	assert.Empty(t, resp.RegistryPassword)
	assert.True(t, resp.HasRegistryPassword)
}

func TestAppResponseJSONNeverSerializesRegistryPassword(t *testing.T) {
	payload, err := json.Marshal(models.AppResponse{
		ID:                  "app-1",
		Name:                "Demo App",
		RegistryPassword:    "should-never-serialize",
		HasRegistryPassword: true,
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	_, exists := decoded["registry_password"]
	assert.False(t, exists)
	assert.Equal(t, true, decoded["has_registry_password"])
}

func TestListAppImageTagsDecryptsRegistryPassword(t *testing.T) {
	setupAppVolumeTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if username != "demo" || password != "super-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)
		case "/v2/demo/tags/list":
			_, err := w.Write([]byte(`{"tags":["v1.0.0"]}`))
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	registryURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	repository := fmt.Sprintf("%s/demo", registryURL.Host)
	encryptedPassword, err := secrets.EncryptString("super-secret")
	require.NoError(t, err)

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
		Base:             entities.Base{ID: "app-1"},
		Slug:             "api",
		Name:             "API",
		EnvID:            "env-1",
		AppType:          app.AppTypeDeployment,
		ContainerImage:   repository + ":v1.0.0",
		RegistryUsername: "demo",
		RegistryPassword: encryptedPassword,
		Replicas:         1,
	}).Error)

	result, err := ListAppImageTags(context.Background(), "app-1")
	require.NoError(t, err)
	assert.Equal(t, repository, result.Repository)
	assert.Equal(t, []string{"v1.0.0"}, result.Tags)
}
