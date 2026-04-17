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

func setupClusterGatewayConfigTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Cluster{}, &entities.ClusterGatewayProvider{}))
	db.DB = testDB
}

func TestUpdateClusterDefaultGatewayClassPersistsConfiguration(t *testing.T) {
	setupClusterGatewayConfigTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-one",
		Name:       "Cluster One",
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)

	_, err := UpdateClusterDefaultGatewayClass("cluster-1", &models.UpdateClusterGatewayClassRequest{
		GatewayClassName:      "ketches-external",
		GatewayControllerName: "gateway.envoyproxy.io/gatewayclass-controller",
		ManagementMode:        "adopted",
	})
	require.NoError(t, err)
	provider, err := GetDefaultClusterGatewayProvider("cluster-1")
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "ketches-external", provider.GatewayClassName)
	assert.Equal(t, "gateway.envoyproxy.io/gatewayclass-controller", provider.ControllerName)
	assert.Equal(t, "adopted", provider.SourceType)
}

func TestUpdateClusterDefaultGatewayClassDefaultsManagementModeToAdopted(t *testing.T) {
	setupClusterGatewayConfigTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-one",
		Name:       "Cluster One",
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)

	_, err := UpdateClusterDefaultGatewayClass("cluster-1", &models.UpdateClusterGatewayClassRequest{
		GatewayClassName: "ketches-external",
	})
	require.NoError(t, err)
	provider, err := GetDefaultClusterGatewayProvider("cluster-1")
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "adopted", provider.SourceType)
}

func TestUpdateClusterDefaultGatewayClassRejectsInvalidManagementMode(t *testing.T) {
	setupClusterGatewayConfigTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-one",
		Name:       "Cluster One",
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)

	updated, err := UpdateClusterDefaultGatewayClass("cluster-1", &models.UpdateClusterGatewayClassRequest{
		GatewayClassName: "ketches-external",
		ManagementMode:   "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "management mode")
}
