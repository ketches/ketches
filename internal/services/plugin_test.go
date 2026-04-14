package services

import (
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
	require.NoError(t, testDB.AutoMigrate(&entities.Plugin{}))
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
	updated, err := UpdatePlugin(plugin.ID, &models.UpdatePluginRequest{
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

	updated, err := UpdatePlugin(plugin.ID, &models.UpdatePluginRequest{
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
	updated, err := UpdatePlugin("plugin-3", &models.UpdatePluginRequest{
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
	updated, err := UpdatePlugin("plugin-4", &models.UpdatePluginRequest{
		ClearRegistryPassword: &clearPassword,
	})
	require.NoError(t, err)
	assert.Empty(t, updated.RegistryPassword)

	var stored entities.Plugin
	require.NoError(t, db.DB.First(&stored, "id = ?", updated.ID).Error)
	assert.Empty(t, stored.RegistryPassword)
}
