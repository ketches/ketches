package services

import (
	"testing"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBuildSettingServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.ContainerRegistry{},
		&entities.BuildSetting{},
	))

	db.DB = testDB
}

func seedBuildSettingRegistry(t *testing.T, registryID string) {
	t.Helper()

	registry := entities.ContainerRegistry{
		ID:        registryID,
		Name:      "Main Registry",
		Provider:  entities.RegistryProviderGHCR,
		Endpoint:  "ghcr.io",
		Scope:     entities.RegistryScopeProject,
		ProjectID: "project-1",
		Enabled:   true,
	}
	require.NoError(t, db.DB.Create(&registry).Error)
}

func TestCreateRepoBuildSetting_DefaultsBuildkitFields(t *testing.T) {
	setupBuildSettingServiceTestDB(t)
	seedBuildSettingRegistry(t, "registry-1")

	setting, err := CreateRepoBuildSetting("repo-1", &models.CreateBuildSettingRequest{
		Name:       "backend",
		ImageName:  "demo/backend",
		RegistryID: "registry-1",
	})
	require.NoError(t, err)

	assert.Equal(t, "linux/amd64", setting.Platforms)
	require.NotNil(t, setting.RegistryCacheEnabled)
	assert.True(t, *setting.RegistryCacheEnabled)
	assert.Empty(t, setting.RegistryCacheRef)
	assert.Equal(t, "main", setting.GitRef)
	assert.Equal(t, "Dockerfile", setting.DockerfilePath)
	assert.Equal(t, ".", setting.BuildContext)
}

func TestCreateRepoBuildSetting_NormalizesStructuredBuildArgs(t *testing.T) {
	setupBuildSettingServiceTestDB(t)
	seedBuildSettingRegistry(t, "registry-1")

	setting, err := CreateRepoBuildSetting("repo-1", &models.CreateBuildSettingRequest{
		Name:       "backend",
		ImageName:  "demo/backend",
		RegistryID: "registry-1",
		BuildArgPairs: []models.BuildArgPair{
			{Key: "ZETA", Value: "last"},
			{Key: "ALPHA", Value: "first"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "ALPHA=first\nZETA=last", setting.BuildArgs)
	resp := ToBuildSettingResponse(setting)
	require.Len(t, resp.BuildArgPairs, 2)
	assert.Equal(t, []models.BuildArgPair{
		{Key: "ALPHA", Value: "first"},
		{Key: "ZETA", Value: "last"},
	}, resp.BuildArgPairs)
}

func TestToBuildSettingResponse_ExposesPlatformsCacheAndArgPairs(t *testing.T) {
	resp := ToBuildSettingResponse(&BuildSettingWithRegistry{
		BuildSetting: entities.BuildSetting{
			ID:                   "setting-1",
			Name:                 "backend",
			GitRef:               "main",
			DockerfilePath:       "deploy/Dockerfile",
			BuildContext:         ".",
			ImageName:            "demo/backend",
			RegistryID:           "registry-1",
			BuildArgs:            "ALPHA=first\nZETA=last",
			Platforms:            "linux/amd64,linux/arm64",
			RegistryCacheEnabled: testBoolPtr(false),
			RegistryCacheRef:     "ghcr.io/demo/backend:buildcache-setting-1",
		},
		RegistrySummary: RegistrySummary{
			RegName:     "Main Registry",
			RegProvider: string(entities.RegistryProviderGHCR),
		},
	})

	assert.Equal(t, "linux/amd64,linux/arm64", resp.Platforms)
	assert.False(t, resp.RegistryCacheEnabled)
	assert.Equal(t, "ghcr.io/demo/backend:buildcache-setting-1", resp.RegistryCacheRef)
	assert.Equal(t, []models.BuildArgPair{
		{Key: "ALPHA", Value: "first"},
		{Key: "ZETA", Value: "last"},
	}, resp.BuildArgPairs)
}

func TestUpdateRepoBuildSetting_PreservesExistingPlatformsWhenOmitted(t *testing.T) {
	setupBuildSettingServiceTestDB(t)
	seedBuildSettingRegistry(t, "registry-1")

	initial := entities.BuildSetting{
		ID:                   "setting-1",
		Name:                 "backend",
		CodeRepositoryID:     ptr("repo-1"),
		GitRef:               "main",
		DockerfilePath:       "Dockerfile",
		BuildContext:         ".",
		ImageName:            "demo/backend",
		RegistryID:           "registry-1",
		BuildArgs:            "ALPHA=first",
		Platforms:            "linux/amd64,linux/arm64",
		RegistryCacheEnabled: testBoolPtr(false),
		RegistryCacheRef:     "ghcr.io/demo/backend:buildcache-setting-1",
	}
	require.NoError(t, db.DB.Select("*").Create(&initial).Error)

	updated, err := UpdateRepoBuildSetting("setting-1", &models.UpdateRepoBuildSettingRequest{
		Name: "backend-updated",
	})
	require.NoError(t, err)

	assert.Equal(t, "backend-updated", updated.Name)
	assert.Equal(t, "linux/amd64,linux/arm64", updated.Platforms)
	require.NotNil(t, updated.RegistryCacheEnabled)
	assert.False(t, *updated.RegistryCacheEnabled)
	assert.Equal(t, "ghcr.io/demo/backend:buildcache-setting-1", updated.RegistryCacheRef)
}

func ptr(v string) *string {
	return &v
}

func testBoolPtr(v bool) *bool {
	return &v
}
