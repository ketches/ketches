package core

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNormalizeReconcileLockDialect(t *testing.T) {
	assert.Equal(t, "postgres", normalizeReconcileLockDialect(" PostgreSQL "))
	assert.Equal(t, "postgres", normalizeReconcileLockDialect("postgres"))
	assert.Equal(t, "mysql", normalizeReconcileLockDialect("MySQL"))
	assert.Empty(t, normalizeReconcileLockDialect("sqlite"))
	assert.Empty(t, normalizeReconcileLockDialect(""))
}

func TestReconcileLockKeysAreStableAndBounded(t *testing.T) {
	postgresKey := postgresReconcileLockID("app:one")
	assert.Equal(t, postgresKey, postgresReconcileLockID("app:one"))
	assert.NotEqual(t, postgresKey, postgresReconcileLockID("app:two"))
	assert.GreaterOrEqual(t, postgresKey, int64(0))

	mysqlKey := mysqlReconcileLockName("app:one")
	assert.Equal(t, mysqlKey, mysqlReconcileLockName("app:one"))
	assert.NotEqual(t, mysqlKey, mysqlReconcileLockName("app:two"))
	assert.LessOrEqual(t, len(mysqlKey), 64)
	assert.NotContains(t, mysqlKey, "app:one")
}

func TestAcquireMySQLReconcileLockRejectsNullResultImmediately(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	connection, err := sqlDB.Conn(context.Background())
	require.NoError(t, err)
	defer connection.Close()

	lockName := mysqlReconcileLockName("app:one")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(?, 0)")).
		WithArgs(lockName).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(nil))

	release, err := acquireDatabaseReconcileLock(context.Background(), connection, "mysql", "app:one")
	require.ErrorContains(t, err, "GET_LOCK returned NULL")
	assert.Nil(t, release)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcquireAndReleaseMySQLReconcileLock(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	connection, err := sqlDB.Conn(context.Background())
	require.NoError(t, err)
	defer connection.Close()

	lockName := mysqlReconcileLockName("app:one")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(?, 0)")).
		WithArgs(lockName).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT RELEASE_LOCK(?)")).
		WithArgs(lockName).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))

	release, err := acquireDatabaseReconcileLock(context.Background(), connection, "mysql", "app:one")
	require.NoError(t, err)
	require.NotNil(t, release)
	require.NoError(t, release())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDatabaseReconcileLockFallsBackForSQLite(t *testing.T) {
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	db.DB = testDB

	called := false
	require.NoError(t, withDatabaseReconcileLock(context.Background(), "app:one", func() error {
		called = true
		return nil
	}))
	assert.True(t, called)
}
