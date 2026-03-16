package services

import (
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
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

func TestListPlatformOperationLogsFiltersPlatformResourceTypes(t *testing.T) {
	setupOperationLogServiceTestDB(t)

	require.NoError(t, CreateOperationLog(CreateOperationLogInput{
		UserID:       "u1",
		Username:     "alice",
		Action:       "update",
		ResourceType: "platform_branding",
		ResourceID:   "platform",
		Status:       entities.OperationLogStatusSuccess,
		StatusCode:   200,
		Sensitivity:  entities.OperationLogSensitivitySensitive,
	}))
	require.NoError(t, CreateOperationLog(CreateOperationLogInput{
		UserID:       "u2",
		Username:     "bob",
		Action:       "deploy",
		ResourceType: "app",
		ResourceID:   "app-1",
		Status:       entities.OperationLogStatusSuccess,
		StatusCode:   200,
		Sensitivity:  entities.OperationLogSensitivityInternal,
	}))

	total, items, err := ListPlatformOperationLogs(models.OperationLogListRequest{
		PaginationRequest: models.PaginationRequest{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "platform_branding", items[0].ResourceType)
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

func TestParseOperationLogTimeSupportsMultipleFormats(t *testing.T) {
	t.Run("RFC3339", func(t *testing.T) {
		parsed, ok := parseOperationLogTime("2026-03-11T09:30:00Z")
		require.True(t, ok)
		assert.Equal(t, 2026, parsed.Year())
		assert.Equal(t, time.March, parsed.Month())
		assert.Equal(t, 11, parsed.Day())
	})

	t.Run("RFC3339 with milliseconds", func(t *testing.T) {
		parsed, ok := parseOperationLogTime("2026-03-11T09:30:00.000Z")
		require.True(t, ok)
		assert.Equal(t, 0, parsed.Nanosecond())
	})

	t.Run("datetime-local minute precision", func(t *testing.T) {
		parsed, ok := parseOperationLogTime("2026-03-11T09:30")
		require.True(t, ok)
		assert.Equal(t, 2026, parsed.Year())
		assert.Equal(t, time.March, parsed.Month())
		assert.Equal(t, 11, parsed.Day())
		assert.Equal(t, 9, parsed.Hour())
		assert.Equal(t, 30, parsed.Minute())
	})

	t.Run("datetime-local second precision", func(t *testing.T) {
		parsed, ok := parseOperationLogTime("2026-03-11T09:30:45")
		require.True(t, ok)
		assert.Equal(t, 45, parsed.Second())
	})

	t.Run("date only", func(t *testing.T) {
		parsed, ok := parseOperationLogTime("2026-03-11")
		require.True(t, ok)
		assert.Equal(t, 2026, parsed.Year())
		assert.Equal(t, time.March, parsed.Month())
		assert.Equal(t, 11, parsed.Day())
		assert.Equal(t, 0, parsed.Hour())
		assert.Equal(t, 0, parsed.Minute())
	})

	t.Run("invalid value", func(t *testing.T) {
		_, ok := parseOperationLogTime("not-a-date")
		assert.False(t, ok)
	})
}

func TestListOperationLogsFiltersDateTimeLocalRange(t *testing.T) {
	setupOperationLogServiceTestDB(t)

	beforeRange := time.Date(2026, time.March, 11, 9, 0, 0, 0, time.Local)
	insideRange := time.Date(2026, time.March, 11, 12, 0, 0, 0, time.Local)
	afterRange := time.Date(2026, time.March, 11, 15, 0, 0, 0, time.Local)

	require.NoError(t, db.DB.Create(&entities.OperationLog{
		Base:         entities.Base{ID: "log-before", CreatedAt: beforeRange},
		Username:     "alice",
		Action:       "deploy",
		ResourceType: "app",
		Status:       entities.OperationLogStatusSuccess,
		StatusCode:   200,
		Sensitivity:  entities.OperationLogSensitivityPublic,
	}).Error)

	require.NoError(t, db.DB.Create(&entities.OperationLog{
		Base:         entities.Base{ID: "log-inside", CreatedAt: insideRange},
		Username:     "bob",
		Action:       "deploy",
		ResourceType: "app",
		Status:       entities.OperationLogStatusSuccess,
		StatusCode:   200,
		Sensitivity:  entities.OperationLogSensitivityPublic,
	}).Error)

	require.NoError(t, db.DB.Create(&entities.OperationLog{
		Base:         entities.Base{ID: "log-after", CreatedAt: afterRange},
		Username:     "charlie",
		Action:       "deploy",
		ResourceType: "app",
		Status:       entities.OperationLogStatusSuccess,
		StatusCode:   200,
		Sensitivity:  entities.OperationLogSensitivityPublic,
	}).Error)

	total, items, err := ListOperationLogs(models.OperationLogListRequest{
		PaginationRequest: models.PaginationRequest{Page: 1, PageSize: 10},
		Start:             "2026-03-11T10:00",
		End:               "2026-03-11T13:00",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "log-inside", items[0].ID)
}

func TestSystemSettingLookupQueryQuotesKeyColumn(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		DryRun:                                   true,
	})
	require.NoError(t, err)

	statement := systemSettingLookupQuery(testDB, "platform_branding").Limit(1).Find(&entities.SystemSetting{}).Statement

	assert.Contains(t, statement.SQL.String(), "`key`")
	assert.NotContains(t, statement.SQL.String(), "WHERE key =")
}
