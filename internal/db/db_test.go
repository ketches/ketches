package db

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInitDB_RejectsSQLiteDriver(t *testing.T) {
	originalConfig := app.Config
	originalDB := DB
	t.Cleanup(func() {
		app.Config = originalConfig
		DB = originalDB
	})

	app.Config = app.AppConfig{
		DBDriver:      "sqlite",
		DBSource:      "file::memory:?cache=shared",
		DBAutoMigrate: false,
	}

	err := InitDB()
	if err == nil {
		t.Fatal("expected sqlite driver to be rejected")
	}
	if err.Error() != "unsupported database driver: sqlite" {
		t.Fatalf("expected unsupported sqlite error, got %v", err)
	}
}

func TestNewGormConfig_DisablesForeignKeyConstraintWhenMigrating(t *testing.T) {
	config := newGormConfig()
	if config == nil {
		t.Fatal("expected gorm config, got nil")
	}

	assert.True(t, config.DisableForeignKeyConstraintWhenMigrating)
}

func TestMigrate_RenamesClusterGatewayAddressColumnToGatewayHost(t *testing.T) {
	originalDB := DB
	t.Cleanup(func() {
		DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), newGormConfig())
	require.NoError(t, err)
	DB = testDB

	require.NoError(t, DB.Exec(`CREATE TABLE clusters (
		id TEXT PRIMARY KEY,
		slug TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		kube_config TEXT NOT NULL,
		api_server TEXT,
		gateway_address TEXT,
		enabled BOOLEAN,
		connection_status TEXT,
		connection_status_reason TEXT,
		last_checked_at TIMESTAMP,
		created_at TIMESTAMP,
		updated_at TIMESTAMP,
		deleted_at TIMESTAMP
	)`).Error)

	require.NoError(t, DB.Exec(`INSERT INTO clusters (id, slug, name, kube_config, gateway_address, enabled) VALUES (?, ?, ?, ?, ?, ?)`,
		"cluster-1", "demo-cluster", "Demo Cluster", "enc:v1:opaque", "gateway.example.com", true,
	).Error)

	require.True(t, DB.Migrator().HasColumn("clusters", "gateway_address"))
	require.False(t, DB.Migrator().HasColumn("clusters", "gateway_host"))

	require.NoError(t, Migrate())

	assert.False(t, DB.Migrator().HasColumn(&entities.Cluster{}, "gateway_address"))
	assert.True(t, DB.Migrator().HasColumn(&entities.Cluster{}, "gateway_host"))

	var cluster entities.Cluster
	require.NoError(t, DB.First(&cluster, "id = ?", "cluster-1").Error)
	assert.Equal(t, "gateway.example.com", cluster.GatewayHost)
}
