package db

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
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

func TestMigrateAppGatewayRoutesCopiesLegacyPublicGateway(t *testing.T) {
	originalDB := DB
	t.Cleanup(func() {
		DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), newGormConfig())
	require.NoError(t, err)
	DB = testDB

	require.NoError(t, DB.Exec(`CREATE TABLE app_gateways (
		id TEXT PRIMARY KEY,
		app_id TEXT NOT NULL,
		port INTEGER NOT NULL,
		protocol TEXT NOT NULL,
		domain TEXT,
		path TEXT,
		gateway_port INTEGER,
		service_type TEXT,
		node_port INTEGER,
		exposed BOOLEAN,
		cert_id TEXT,
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	)`).Error)
	require.NoError(t, DB.Exec(`INSERT INTO app_gateways (
		id, app_id, port, protocol, domain, path, gateway_port, service_type, exposed, cert_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"gateway-1", "app-1", 8080, "https", "api.example.com", "/api", 0, "ClusterIP", true, "cert-1",
	).Error)

	require.NoError(t, Migrate())

	var routes []entities.AppGatewayHTTPRoute
	require.NoError(t, DB.Find(&routes).Error)
	require.Len(t, routes, 1)
	assert.Equal(t, "gateway-1", routes[0].AppGatewayID)
	assert.Equal(t, "api.example.com", routes[0].Host)
	assert.Equal(t, "https", routes[0].ListenerProtocol)
	assert.Equal(t, "/api", routes[0].Path)
	assert.Equal(t, "PathPrefix", routes[0].PathMatchType)
	assert.True(t, routes[0].Enabled)
	require.NotNil(t, routes[0].CertID)
	assert.Equal(t, "cert-1", *routes[0].CertID)

	var backends []entities.AppGatewayHTTPRouteBackend
	require.NoError(t, DB.Find(&backends).Error)
	require.Len(t, backends, 1)
	assert.Equal(t, routes[0].ID, backends[0].RouteID)
	assert.Equal(t, "app-1", backends[0].BackendAppID)
	assert.Equal(t, 8080, backends[0].BackendPort)
	assert.Equal(t, 1, backends[0].Weight)
}
