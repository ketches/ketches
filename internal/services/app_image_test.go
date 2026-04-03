package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	internalapp "github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
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

func TestUpdateAppImageMigratesLegacyPlaintextRegistryPassword(t *testing.T) {
	setupAppVolumeTestDB(t)
	require.NoError(t, db.DB.AutoMigrate(&entities.DeploymentHistory{}))

	originalKey := internalapp.Config.SecretEncryptionKey
	internalapp.Config.SecretEncryptionKey = "test-master-key"
	t.Cleanup(func() {
		internalapp.Config.SecretEncryptionKey = originalKey
	})

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
		AppType:          "Deployment",
		ContainerImage:   "nginx:1.25",
		RegistryUsername: "robot",
		RegistryPassword: "legacy-plaintext-password",
		Replicas:         1,
	}).Error)

	originalApplyAppFn := applyAppFn
	applyAppFn = func(_ context.Context, appCtx *models.AppContext) error {
		decryptedPassword, err := secrets.DecryptString(appCtx.App.RegistryPassword)
		require.NoError(t, err)
		assert.Equal(t, "legacy-plaintext-password", decryptedPassword)
		return nil
	}
	t.Cleanup(func() {
		applyAppFn = originalApplyAppFn
	})

	appCtx, err := UpdateAppImage(context.Background(), "app-1", &models.UpdateAppImageRequest{
		ContainerImage: "nginx:1.26",
	})

	require.NoError(t, err)
	require.NotNil(t, appCtx)
	assert.True(t, strings.HasPrefix(appCtx.App.RegistryPassword, "enc:v1:"))

	var stored entities.App
	require.NoError(t, db.DB.First(&stored, "id = ?", "app-1").Error)
	assert.True(t, strings.HasPrefix(stored.RegistryPassword, "enc:v1:"))

	decryptedPassword, err := secrets.DecryptString(stored.RegistryPassword)
	require.NoError(t, err)
	assert.Equal(t, "legacy-plaintext-password", decryptedPassword)
}
