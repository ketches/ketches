package services

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReconcilePublicGatewayResourcesIncludesEnabledClustersWithoutRoutes(t *testing.T) {
	originalDB := db.DB
	originalEnsure := ensureSharedGatewayForReconcile
	t.Cleanup(func() {
		db.DB = originalDB
		ensureSharedGatewayForReconcile = originalEnsure
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Cluster{}))
	require.NoError(t, testDB.Create(&entities.Cluster{
		Base: entities.Base{ID: "cluster-enabled"}, Slug: "enabled", Name: "Enabled",
		KubeConfig: "kubeconfig", Enabled: true,
	}).Error)
	require.NoError(t, testDB.Create(&entities.Cluster{
		Base: entities.Base{ID: "cluster-disabled"}, Slug: "disabled", Name: "Disabled",
		KubeConfig: "kubeconfig",
	}).Error)
	require.NoError(t, testDB.Model(&entities.Cluster{}).
		Where("id = ?", "cluster-disabled").Update("enabled", false).Error)
	db.DB = testDB

	var reconciled []string
	ensureSharedGatewayForReconcile = func(_ context.Context, clusterID string) error {
		reconciled = append(reconciled, clusterID)
		return nil
	}

	require.NoError(t, ReconcilePublicGatewayResources(context.Background()))
	assert.Equal(t, []string{"cluster-enabled"}, reconciled)
}
