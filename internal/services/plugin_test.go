package services

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPluginServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	db.DB = testDB
	require.NoError(t, testDB.AutoMigrate(&entities.Plugin{}, &entities.AppPlugin{}))
}

func TestPluginOperationsRequireMatchingParent(t *testing.T) {
	setupPluginServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Plugin{
		ID:         "plugin-2",
		ProjectID:  "project-2",
		Slug:       "plugin-two",
		Name:       "Other Project",
		Image:      "docker.io/library/busybox:latest",
		PluginType: "init",
	}).Error)

	_, err := GetPlugin("project-1", "plugin-2")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	_, err = UpdatePlugin("project-1", "plugin-2", &models.UpdatePluginRequest{Name: "Tampered"})
	assert.Error(t, err)

	require.Error(t, DeletePlugin("project-1", "plugin-2"))
	var stored entities.Plugin
	require.NoError(t, db.DB.First(&stored, "id = ?", "plugin-2").Error)
	assert.Equal(t, "Other Project", stored.Name)
}

func TestUpdatePluginClearsRegistryUsernameWhenExplicitlyEmpty(t *testing.T) {
	setupPluginServiceTestDB(t)

	plugin := &entities.Plugin{
		ID:               "plugin-1",
		ProjectID:        "project-1",
		Slug:             "plugin-one",
		Name:             "Plugin One",
		Image:            "docker.io/library/busybox:latest",
		RegistryUsername: "robot",
		PluginType:       "init",
	}
	require.NoError(t, db.DB.Create(plugin).Error)

	emptyUsername := ""
	updated, err := UpdatePlugin("project-1", plugin.ID, &models.UpdatePluginRequest{
		RegistryUsername: &emptyUsername,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)

	var stored entities.Plugin
	require.NoError(t, db.DB.First(&stored, "id = ?", plugin.ID).Error)
	assert.Equal(t, "", stored.RegistryUsername)
}

func TestUpdatePluginKeepsRegistryUsernameWhenFieldIsOmitted(t *testing.T) {
	setupPluginServiceTestDB(t)

	plugin := &entities.Plugin{
		ID:               "plugin-2",
		ProjectID:        "project-1",
		Slug:             "plugin-two",
		Name:             "Plugin Two",
		Image:            "docker.io/library/busybox:latest",
		RegistryUsername: "robot",
		PluginType:       "init",
	}
	require.NoError(t, db.DB.Create(plugin).Error)

	updated, err := UpdatePlugin("project-1", plugin.ID, &models.UpdatePluginRequest{
		Name: "Plugin Two Updated",
	})
	require.NoError(t, err)
	require.NotNil(t, updated)

	var stored entities.Plugin
	require.NoError(t, db.DB.First(&stored, "id = ?", plugin.ID).Error)
	assert.Equal(t, "robot", stored.RegistryUsername)
	assert.Equal(t, "Plugin Two Updated", stored.Name)
}

func TestCreatePluginEncryptsRegistryPasswordAtRest(t *testing.T) {
	setupPluginServiceTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	plugin, err := CreatePlugin(&models.CreatePluginRequest{
		ProjectID:        "project-1",
		Slug:             "plugin-one",
		Name:             "Plugin One",
		Image:            "docker.io/library/busybox:latest",
		RegistryUsername: "robot",
		RegistryPassword: "super-secret",
		PluginType:       "init",
	})
	require.NoError(t, err)

	assert.NotEqual(t, "super-secret", plugin.RegistryPassword)
	assert.Contains(t, plugin.RegistryPassword, "enc:v1:")

	var stored entities.Plugin
	require.NoError(t, db.DB.First(&stored, "id = ?", plugin.ID).Error)
	assert.Contains(t, stored.RegistryPassword, "enc:v1:")

	decrypted, err := secrets.DecryptString(stored.RegistryPassword)
	require.NoError(t, err)
	assert.Equal(t, "super-secret", decrypted)
}

func TestUpdatePluginEncryptsRegistryPasswordAtRest(t *testing.T) {
	setupPluginServiceTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	require.NoError(t, db.DB.Create(&entities.Plugin{
		ID:         "plugin-3",
		ProjectID:  "project-1",
		Slug:       "plugin-three",
		Name:       "Plugin Three",
		Image:      "docker.io/library/busybox:latest",
		PluginType: "init",
	}).Error)

	password := "new-super-secret"
	updated, err := UpdatePlugin("project-1", "plugin-3", &models.UpdatePluginRequest{
		RegistryPassword: &password,
	})
	require.NoError(t, err)

	assert.NotEqual(t, password, updated.RegistryPassword)
	assert.Contains(t, updated.RegistryPassword, "enc:v1:")

	var stored entities.Plugin
	require.NoError(t, db.DB.First(&stored, "id = ?", updated.ID).Error)
	assert.Contains(t, stored.RegistryPassword, "enc:v1:")

	decrypted, err := secrets.DecryptString(stored.RegistryPassword)
	require.NoError(t, err)
	assert.Equal(t, "new-super-secret", decrypted)
}

func TestUpdatePluginClearsRegistryPasswordAtRest(t *testing.T) {
	setupPluginServiceTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedPassword, err := secrets.EncryptString("old-secret")
	require.NoError(t, err)
	require.NoError(t, db.DB.Create(&entities.Plugin{
		ID:               "plugin-4",
		ProjectID:        "project-1",
		Slug:             "plugin-four",
		Name:             "Plugin Four",
		Image:            "docker.io/library/busybox:latest",
		RegistryPassword: encryptedPassword,
		PluginType:       "init",
	}).Error)

	clearPassword := true
	updated, err := UpdatePlugin("project-1", "plugin-4", &models.UpdatePluginRequest{
		ClearRegistryPassword: &clearPassword,
	})
	require.NoError(t, err)
	assert.Empty(t, updated.RegistryPassword)

	var stored entities.Plugin
	require.NoError(t, db.DB.First(&stored, "id = ?", updated.ID).Error)
	assert.Empty(t, stored.RegistryPassword)
}

func TestUpdateAppPluginResourcesUpdatesRecordAndAppliesApp(t *testing.T) {
	setupAppVolumeTestDB(t)
	require.NoError(t, db.DB.AutoMigrate(&entities.Plugin{}))

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
		Slug:           "demo-app",
		Name:           "Demo App",
		EnvID:          "env-1",
		AppType:        "Deployment",
		ContainerImage: "nginx:1.27",
		Replicas:       1,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Plugin{
		ID:         "plugin-1",
		ProjectID:  "project-1",
		Slug:       "sidecar-one",
		Name:       "Sidecar One",
		Image:      "busybox:1.36",
		PluginType: "sidecar",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppPlugin{
		ID:       "app-plugin-1",
		AppID:    "app-1",
		PluginID: "plugin-1",
		Enabled:  true,
	}).Error)

	originalApplyAppFn := applyAppFn
	applied := false
	applyAppFn = func(_ context.Context, appCtx *models.AppContext) error {
		applied = true
		require.Len(t, appCtx.AppPlugins, 1)
		assert.Equal(t, 200, appCtx.AppPlugins[0].RequestCPU)
		assert.Equal(t, 256, appCtx.AppPlugins[0].RequestMemory)
		assert.Equal(t, 500, appCtx.AppPlugins[0].LimitCPU)
		assert.Equal(t, 512, appCtx.AppPlugins[0].LimitMemory)
		return nil
	}
	t.Cleanup(func() {
		applyAppFn = originalApplyAppFn
	})

	err := UpdateAppPluginResources(context.Background(), "app-1", "plugin-1", &models.UpdateAppPluginResourcesRequest{
		RequestCPU:    200,
		RequestMemory: 256,
		LimitCPU:      500,
		LimitMemory:   512,
	})
	require.NoError(t, err)
	assert.True(t, applied)

	var stored entities.AppPlugin
	require.NoError(t, db.DB.First(&stored, "id = ?", "app-plugin-1").Error)
	assert.Equal(t, 200, stored.RequestCPU)
	assert.Equal(t, 256, stored.RequestMemory)
	assert.Equal(t, 500, stored.LimitCPU)
	assert.Equal(t, 512, stored.LimitMemory)
}

func TestInstallPluginToAppAppliesDefaultResources(t *testing.T) {
	setupAppVolumeTestDB(t)
	require.NoError(t, db.DB.AutoMigrate(&entities.Plugin{}))

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
		Slug:           "demo-app",
		Name:           "Demo App",
		EnvID:          "env-1",
		AppType:        "Deployment",
		ContainerImage: "nginx:1.27",
		Replicas:       1,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Plugin{
		ID:         "plugin-1",
		ProjectID:  "project-1",
		Slug:       "sidecar-one",
		Name:       "Sidecar One",
		Image:      "busybox:1.36",
		PluginType: "sidecar",
	}).Error)

	appPlugin, err := InstallPluginToApp("app-1", &models.InstallPluginRequest{
		PluginID: "plugin-1",
	})
	require.NoError(t, err)
	require.NotNil(t, appPlugin)
	assert.Equal(t, entities.DefaultAppPluginRequestCPU, appPlugin.RequestCPU)
	assert.Equal(t, entities.DefaultAppPluginRequestMemory, appPlugin.RequestMemory)
	assert.Equal(t, entities.DefaultAppPluginLimitCPU, appPlugin.LimitCPU)
	assert.Equal(t, entities.DefaultAppPluginLimitMemory, appPlugin.LimitMemory)

	var stored entities.AppPlugin
	require.NoError(t, db.DB.First(&stored, "id = ?", appPlugin.ID).Error)
	assert.Equal(t, entities.DefaultAppPluginRequestCPU, stored.RequestCPU)
	assert.Equal(t, entities.DefaultAppPluginRequestMemory, stored.RequestMemory)
	assert.Equal(t, entities.DefaultAppPluginLimitCPU, stored.LimitCPU)
	assert.Equal(t, entities.DefaultAppPluginLimitMemory, stored.LimitMemory)
}
