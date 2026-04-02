package services

import (
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupContainerRegistrySecretTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.ContainerRegistry{}))

	db.DB = testDB
}

func TestCreateClusterRegistryEncryptsPassword(t *testing.T) {
	setupContainerRegistrySecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	registry, err := CreateClusterRegistry("cluster-1", &models.CreateContainerRegistryRequest{
		Name:     "Main Registry",
		Provider: string(entities.RegistryProviderGHCR),
		Endpoint: "ghcr.io",
		Username: "demo",
		Password: "super-secret",
		Enabled:  true,
	})
	require.NoError(t, err)

	assert.NotEqual(t, "super-secret", registry.Password)

	var stored entities.ContainerRegistry
	require.NoError(t, db.DB.First(&stored, "id = ?", registry.ID).Error)
	assert.NotEqual(t, "super-secret", stored.Password)
}

func TestUpdateContainerRegistryEncryptsPassword(t *testing.T) {
	setupContainerRegistrySecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	require.NoError(t, db.DB.Create(&entities.ContainerRegistry{
		ID:        "registry-1",
		Name:      "Main Registry",
		Provider:  entities.RegistryProviderGHCR,
		Endpoint:  "ghcr.io",
		Scope:     entities.RegistryScopeCluster,
		ClusterID: registryStringPtr("cluster-1"),
		Enabled:   true,
	}).Error)

	registry, err := UpdateContainerRegistry("registry-1", &models.UpdateContainerRegistryRequest{
		Password: "new-secret",
	})
	require.NoError(t, err)
	assert.NotEqual(t, "new-secret", registry.Password)

	var stored entities.ContainerRegistry
	require.NoError(t, db.DB.First(&stored, "id = ?", registry.ID).Error)
	assert.NotEqual(t, "new-secret", stored.Password)
}

func TestToContainerRegistryResponseOmitsPassword(t *testing.T) {
	res := ToContainerRegistryResponse(&entities.ContainerRegistry{
		ID:       "registry-1",
		Name:     "Main Registry",
		Provider: entities.RegistryProviderGHCR,
		Password: "enc:v1:opaque",
	})

	assert.Empty(t, res.Password)
	assert.True(t, res.HasPassword)
}

func TestContainerRegistryResponseJSONNeverSerializesPassword(t *testing.T) {
	payload, err := json.Marshal(models.ContainerRegistryResponse{
		ID:          "registry-1",
		Name:        "Main Registry",
		Password:    "should-never-serialize",
		HasPassword: true,
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	_, exists := decoded["password"]
	assert.False(t, exists)
	assert.Equal(t, true, decoded["has_password"])
}

func registryStringPtr(v string) *string {
	return &v
}
