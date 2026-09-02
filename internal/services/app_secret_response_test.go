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

func TestViewerAppConfigurationResponsesOnlyExposeValueStatus(t *testing.T) {
	setupAppVolumeTestDB(t)
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.SecretEncryptionKey = "app-configuration-test-key"

	encryptedEnvValue, err := secrets.EncryptString("secret-env-value")
	require.NoError(t, err)
	encryptedContent, err := secrets.EncryptString("secret-file-content")
	require.NoError(t, err)
	require.NoError(t, db.DB.Create([]entities.AppEnvVar{
		{ID: "env-public", AppID: "app-1", Key: "PUBLIC_URL", Value: "https://example.com"},
		{ID: "env-secret", AppID: "app-1", Key: "API_TOKEN", Value: encryptedEnvValue, IsSecret: true},
	}).Error)
	require.NoError(t, db.DB.Create([]entities.AppConfigFile{
		{ID: "config-public", AppID: "app-1", Slug: "app.conf", MountPath: "/etc/app.conf", Content: "debug=false"},
		{ID: "config-secret", AppID: "app-1", Slug: "credentials", MountPath: "/etc/credentials", Content: encryptedContent, IsSecret: true},
	}).Error)

	envVars, err := ListAppEnvVarsForProjectRole("app-1", app.ProjectRoleViewer)
	require.NoError(t, err)
	require.Len(t, envVars, 2)
	for _, envVar := range envVars {
		assert.Empty(t, envVar.Value)
		assert.True(t, envVar.HasValue)
	}

	configFiles, err := ListAppConfigFilesForProjectRole("app-1", app.ProjectRoleViewer)
	require.NoError(t, err)
	require.Len(t, configFiles, 2)
	for _, configFile := range configFiles {
		assert.Empty(t, configFile.Content)
		assert.True(t, configFile.HasValue)
	}

	developerEnvVars, err := ListAppEnvVarsForProjectRole("app-1", app.ProjectRoleDeveloper)
	require.NoError(t, err)
	values := make(map[string]string, len(developerEnvVars))
	for _, envVar := range developerEnvVars {
		values[envVar.Key] = envVar.Value
	}
	assert.Equal(t, "https://example.com", values["PUBLIC_URL"])
	assert.Equal(t, "secret-env-value", values["API_TOKEN"])
}

func seedAppSecretResponseTestData(t *testing.T) {
	t.Helper()

	encryptedEnvValue, err := secrets.EncryptString("secret-env-value")
	require.NoError(t, err)
	encryptedContent, err := secrets.EncryptString("secret-file-content")
	require.NoError(t, err)
	require.NoError(t, db.DB.Create([]entities.AppEnvVar{
		{ID: "env-public", AppID: "app-1", Key: "PUBLIC_URL", Value: "https://example.com"},
		{ID: "env-secret", AppID: "app-1", Key: "API_TOKEN", Value: encryptedEnvValue, IsSecret: true},
	}).Error)
	require.NoError(t, db.DB.Create([]entities.AppConfigFile{
		{ID: "config-public", AppID: "app-1", Slug: "app.conf", MountPath: "/etc/app.conf", Content: "debug=false"},
		{ID: "config-secret", AppID: "app-1", Slug: "credentials", MountPath: "/etc/credentials", Content: encryptedContent, IsSecret: true},
	}).Error)
}

func TestListAppConfigurationForProjectRoleFailsClosed(t *testing.T) {
	setupAppVolumeTestDB(t)
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.SecretEncryptionKey = "app-configuration-role-test-key"
	seedAppSecretResponseTestData(t)

	tests := []struct {
		name   string
		role   app.ProjectRole
		reveal bool
	}{
		{name: "owner", role: app.ProjectRoleOwner, reveal: true},
		{name: "developer", role: app.ProjectRoleDeveloper, reveal: true},
		{name: "viewer", role: app.ProjectRoleViewer},
		{name: "missing", role: ""},
		{name: "unknown", role: "auditor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars, err := ListAppEnvVarsForProjectRole("app-1", tt.role)
			require.NoError(t, err)
			require.Len(t, envVars, 2)
			configFiles, err := ListAppConfigFilesForProjectRole("app-1", tt.role)
			require.NoError(t, err)
			require.Len(t, configFiles, 2)

			for _, envVar := range envVars {
				assert.Equal(t, tt.reveal && envVar.Key == "API_TOKEN", envVar.Value == "secret-env-value")
				if !tt.reveal {
					assert.Empty(t, envVar.Value)
				}
				assert.True(t, envVar.HasValue)
			}
			for _, configFile := range configFiles {
				if tt.reveal {
					if configFile.Slug == "credentials" {
						assert.Equal(t, "secret-file-content", configFile.Content)
					} else {
						assert.Equal(t, "debug=false", configFile.Content)
					}
				} else {
					assert.Empty(t, configFile.Content)
				}
				assert.True(t, configFile.HasValue)
			}
		})
	}
}

func TestListAppConfigurationCompatibilityWrappersRedactByDefault(t *testing.T) {
	setupAppVolumeTestDB(t)
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.SecretEncryptionKey = "app-configuration-wrapper-test-key"
	seedAppSecretResponseTestData(t)

	envVars, err := ListAppEnvVars("app-1")
	require.NoError(t, err)
	for _, envVar := range envVars {
		assert.Empty(t, envVar.Value)
	}

	configFiles, err := ListAppConfigFiles("app-1")
	require.NoError(t, err)
	for _, configFile := range configFiles {
		assert.Empty(t, configFile.Content)
	}
}
