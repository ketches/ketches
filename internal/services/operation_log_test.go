package services

import (
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOperationLogServiceTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.OperationLog{}, &entities.SystemSetting{}))
	db.DB = testDB
}

func TestOperationLogRetentionDefaultAndUpdate(t *testing.T) {
	setupOperationLogServiceTestDB(t)

	days, err := GetOperationLogRetentionDays()
	require.NoError(t, err)
	assert.Equal(t, 90, days)

	err = UpdateOperationLogRetentionDays(0)
	assert.Error(t, err)

	err = UpdateOperationLogRetentionDays(180)
	require.NoError(t, err)

	days, err = GetOperationLogRetentionDays()
	require.NoError(t, err)
	assert.Equal(t, 180, days)
}

func TestListOperationLogsFilters(t *testing.T) {
	setupOperationLogServiceTestDB(t)

	require.NoError(t, CreateOperationLog(CreateOperationLogInput{UserID: "u1", Username: "alice", Action: "deploy", ResourceType: "app", ResourceID: "a1", Status: entities.OperationLogStatusSuccess, StatusCode: 200, Sensitivity: entities.OperationLogSensitivityPublic}))
	require.NoError(t, CreateOperationLog(CreateOperationLogInput{UserID: "u2", Username: "bob", Action: "delete", ResourceType: "app", ResourceID: "a2", Status: entities.OperationLogStatusFailure, StatusCode: 500, Sensitivity: entities.OperationLogSensitivitySensitive}))

	total, items, err := ListOperationLogs(models.OperationLogListRequest{PaginationRequest: models.PaginationRequest{Page: 1, PageSize: 10}, Action: "delete", Status: entities.OperationLogStatusFailure})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "bob", items[0].Username)

	total, items, err = ListActivities(models.OperationLogListRequest{PaginationRequest: models.PaginationRequest{Page: 1, PageSize: 10}}, "u1", false)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "alice", items[0].Username)
}

func TestCleanupExpiredOperationLogs(t *testing.T) {
	setupOperationLogServiceTestDB(t)

	require.NoError(t, UpdateOperationLogRetentionDays(1))

	oldCreatedAt := time.Now().Add(-48 * time.Hour)
	recentCreatedAt := time.Now().Add(-2 * time.Hour)

	require.NoError(t, db.DB.Create(&entities.OperationLog{Base: entities.Base{ID: "old", CreatedAt: oldCreatedAt}, Action: "deploy", ResourceType: "app", Status: entities.OperationLogStatusSuccess, StatusCode: 200, Sensitivity: entities.OperationLogSensitivityPublic}).Error)
	require.NoError(t, db.DB.Create(&entities.OperationLog{Base: entities.Base{ID: "new", CreatedAt: recentCreatedAt}, Action: "deploy", ResourceType: "app", Status: entities.OperationLogStatusSuccess, StatusCode: 200, Sensitivity: entities.OperationLogSensitivityPublic}).Error)

	deleted, err := CleanupExpiredOperationLogs()
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	var count int64
	require.NoError(t, db.DB.Model(&entities.OperationLog{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
