package db

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestInitDB_RejectsSQLiteDriver(t *testing.T) {
	originalConfig := app.Config
	originalDB := DB
	t.Cleanup(func() {
		app.Config = originalConfig
		DB = originalDB
	})

	app.Config = app.AppConfig{
		DBDriver: "sqlite",
		DBSource: "file::memory:?cache=shared",
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

func TestMigrateConfiguredDatabaseSkipsSchemaChangesWhenDisabled(t *testing.T) {
	testDB := useSQLiteMigrationTestDB(t)
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.DBAutoMigrate = false

	if err := migrateConfiguredDatabase(); err != nil {
		t.Fatalf("skip configured migration: %v", err)
	}
	if testDB.Migrator().HasTable("users") {
		t.Fatal("expected DB_AUTO_MIGRATE=false to leave the schema unchanged")
	}
}

func TestMigrateConfiguredDatabaseAppliesSchemaWhenEnabled(t *testing.T) {
	testDB := useSQLiteMigrationTestDB(t)
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.DBAutoMigrate = true

	if err := migrateConfiguredDatabase(); err != nil {
		t.Fatalf("run configured migration: %v", err)
	}
	if !testDB.Migrator().HasTable("users") {
		t.Fatal("expected DB_AUTO_MIGRATE=true to apply the schema")
	}
}
