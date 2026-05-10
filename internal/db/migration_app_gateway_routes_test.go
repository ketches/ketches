package db

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAppGatewayRouteMigrationCreatesTablesWithoutForeignKeys(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), newGormConfig())
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.AppGateway{},
		&entities.AppGatewayHTTPRoute{},
		&entities.AppGatewayHTTPRouteBackend{},
	))

	require.True(t, testDB.Migrator().HasTable(&entities.AppGatewayHTTPRoute{}))
	require.True(t, testDB.Migrator().HasTable(&entities.AppGatewayHTTPRouteBackend{}))

	var routeFKs []struct {
		ID string
	}
	require.NoError(t, testDB.Raw("PRAGMA foreign_key_list(app_gateway_http_routes)").Scan(&routeFKs).Error)
	require.Empty(t, routeFKs)

	var backendFKs []struct {
		ID string
	}
	require.NoError(t, testDB.Raw("PRAGMA foreign_key_list(app_gateway_http_route_backends)").Scan(&backendFKs).Error)
	require.Empty(t, backendFKs)
}
