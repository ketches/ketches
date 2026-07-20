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

func TestMigrateRequiresLegacyBootstrapAdminPasswordChange(t *testing.T) {
	originalDB := DB
	t.Cleanup(func() { DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.Exec(`CREATE TABLE users (
		id text PRIMARY KEY,
		username text,
		email text,
		password text,
		role text
	)`).Error)
	require.NoError(t, testDB.Exec(`INSERT INTO users (id, username, email, password, role) VALUES
		('bootstrap-admin', 'kadmin', 'kadmin@local.ketches', 'old-hash', 'admin'),
		('regular-admin', 'admin', 'admin@example.com', 'hash', 'admin'),
		('local-user', 'local-user', 'local-user@local.ketches', 'hash', 'user')`).Error)

	DB = testDB
	require.NoError(t, Migrate())

	var bootstrapAdmin entities.User
	require.NoError(t, DB.First(&bootstrapAdmin, "id = ?", "bootstrap-admin").Error)
	assert.True(t, bootstrapAdmin.MustChangePassword)

	var regularAdmin entities.User
	require.NoError(t, DB.First(&regularAdmin, "id = ?", "regular-admin").Error)
	assert.False(t, regularAdmin.MustChangePassword)

	var localUser entities.User
	require.NoError(t, DB.First(&localUser, "id = ?", "local-user").Error)
	assert.Equal(t, app.UserRoleUser, localUser.Role)
	assert.False(t, localUser.MustChangePassword)
}
