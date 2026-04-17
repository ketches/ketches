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

func setupClusterGatewayProviderTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Cluster{}, &entities.ClusterGatewayProvider{}))
	db.DB = testDB
}

func TestRegisterAdoptedGatewayProviderMarksFirstProviderDefault(t *testing.T) {
	setupClusterGatewayProviderTestDB(t)
	require.NoError(t, db.DB.Create(&entities.Cluster{Base: entities.Base{ID: "cluster-1"}, Slug: "cluster-one", Name: "Cluster One", KubeConfig: "enc:v1:test", Enabled: true}).Error)
	provider, err := RegisterAdoptedGatewayProvider("cluster-1", "ketches-existing", "example.com/controller")
	require.NoError(t, err)
	assert.True(t, provider.IsDefault)
}

func TestSetDefaultClusterGatewayProviderClearsPreviousDefault(t *testing.T) {
	setupClusterGatewayProviderTestDB(t)
	require.NoError(t, db.DB.Create(&entities.Cluster{Base: entities.Base{ID: "cluster-1"}, Slug: "cluster-one", Name: "Cluster One", KubeConfig: "enc:v1:test", Enabled: true}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterGatewayProvider{ID: "provider-1", ClusterID: "cluster-1", SourceType: "adopted", DisplayName: "one", GatewayClassName: "one", ControllerName: "controller/one", IsDefault: true}).Error)
	_, err := SetDefaultClusterGatewayProvider("cluster-1", "two", "controller/two", "adopted")
	require.NoError(t, err)
	var providers []entities.ClusterGatewayProvider
	require.NoError(t, db.DB.Where("cluster_id = ?", "cluster-1").Order("gateway_class_name").Find(&providers).Error)
	require.Len(t, providers, 2)
	assert.False(t, providers[0].IsDefault)
	assert.True(t, providers[1].IsDefault)
}

func TestCreateClusterGatewayProviderDoesNotSetDefaultWhenNotRequested(t *testing.T) {
	setupClusterGatewayProviderTestDB(t)
	require.NoError(t, db.DB.Create(&entities.Cluster{Base: entities.Base{ID: "cluster-1"}, Slug: "cluster-one", Name: "Cluster One", KubeConfig: "enc:v1:test", Enabled: true}).Error)

	provider, err := CreateClusterGatewayProvider("cluster-1", &models.CreateClusterGatewayProviderRequest{
		DisplayName:      "Existing Gateway",
		GatewayClassName: "existing-gateway",
		ControllerName:   "example.com/controller",
		MakeDefault:      false,
	})
	require.NoError(t, err)
	assert.False(t, provider.IsDefault)
}

func TestDeleteClusterGatewayProviderRejectsDefaultProvider(t *testing.T) {
	setupClusterGatewayProviderTestDB(t)
	require.NoError(t, db.DB.Create(&entities.ClusterGatewayProvider{ID: "provider-1", ClusterID: "cluster-1", SourceType: "adopted", DisplayName: "one", GatewayClassName: "one", ControllerName: "controller/one", IsDefault: true}).Error)
	err := DeleteClusterGatewayProvider("cluster-1", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default gateway provider")
}

func TestDeleteClusterGatewayProviderRejectsManagedProvider(t *testing.T) {
	setupClusterGatewayProviderTestDB(t)
	require.NoError(t, db.DB.Create(&entities.ClusterGatewayProvider{ID: "provider-1", ClusterID: "cluster-1", SourceType: "managed", DisplayName: "one", GatewayClassName: "one", ControllerName: "controller/one", IsDefault: false}).Error)
	err := DeleteClusterGatewayProvider("cluster-1", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adopted gateway providers")
}
