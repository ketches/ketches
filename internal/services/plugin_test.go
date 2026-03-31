package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
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
		Image:            "docker.io/library/migrate:latest",
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
		Image:            "docker.io/library/migrate:latest",
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
