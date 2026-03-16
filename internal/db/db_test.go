package db

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
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
