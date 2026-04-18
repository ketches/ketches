package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAppEnvVarTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/app-env-var-test.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(&entities.AppEnvVar{}))
	db.DB = testDB
}

func TestCreateAppEnvVarCreatesRecord(t *testing.T) {
	setupAppEnvVarTestDB(t)

	result, err := CreateAppEnvVar("app-1", "MYSQL_ROOT_PASSWORD", "secret")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "app-1", result.AppID)
	assert.Equal(t, "MYSQL_ROOT_PASSWORD", result.Key)
	assert.Equal(t, "secret", result.Value)

	var stored entities.AppEnvVar
	require.NoError(t, db.DB.Where("id = ?", result.ID).First(&stored).Error)
	assert.Equal(t, "MYSQL_ROOT_PASSWORD", stored.Key)
}

func TestCreateAppEnvVarRejectsDuplicateKeyForSameApp(t *testing.T) {
	setupAppEnvVarTestDB(t)

	require.NoError(t, db.DB.Create(&entities.AppEnvVar{
		ID:    "env-1",
		AppID: "app-1",
		Key:   "MYSQL_ROOT_PASSWORD",
		Value: "secret",
	}).Error)

	result, err := CreateAppEnvVar("app-1", "MYSQL_ROOT_PASSWORD", "new-secret")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already exists")
}
