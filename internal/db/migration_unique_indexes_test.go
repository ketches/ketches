package db

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func useSQLiteMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalDB := DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	testDB, err := gorm.Open(sqlite.Open(dsn), newGormConfig())
	require.NoError(t, err)
	sqlDB, err := testDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		DB = originalDB
		require.NoError(t, sqlDB.Close())
	})
	DB = testDB
	return testDB
}

func TestMigrateDeduplicatesLegacyProjectMembersBeforeCreatingUniqueIndex(t *testing.T) {
	testDB := useSQLiteMigrationTestDB(t)
	require.NoError(t, testDB.AutoMigrate(&entities.ProjectMember{}))
	require.NoError(t, testDB.Migrator().DropIndex(&entities.ProjectMember{}, "idx_project_members_project_user"))

	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, testDB.Create(&entities.ProjectMember{
		ID:          "member-a",
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
		ProjectID:   "project-1",
		UserID:      "user-1",
		ProjectRole: app.ProjectRoleViewer,
	}).Error)
	require.NoError(t, testDB.Create(&entities.ProjectMember{
		ID:          "member-z",
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
		ProjectID:   "project-1",
		UserID:      "user-1",
		ProjectRole: app.ProjectRoleOwner,
	}).Error)

	require.NoError(t, Migrate())

	var members []entities.ProjectMember
	require.NoError(t, testDB.Where("project_id = ? AND user_id = ?", "project-1", "user-1").Find(&members).Error)
	require.Len(t, members, 1)
	assert.Equal(t, "member-z", members[0].ID)
	assert.Equal(t, app.ProjectRoleOwner, members[0].ProjectRole)
	assert.True(t, testDB.Migrator().HasIndex(&entities.ProjectMember{}, "idx_project_members_project_user"))

	duplicate := entities.ProjectMember{
		ID:          "member-c",
		ProjectID:   "project-1",
		UserID:      "user-1",
		ProjectRole: app.ProjectRoleDeveloper,
	}
	assert.Error(t, testDB.Create(&duplicate).Error)
}

func TestMigrateNormalizesAndDeduplicatesLegacySignupVerificationCodes(t *testing.T) {
	testDB := useSQLiteMigrationTestDB(t)
	require.NoError(t, testDB.AutoMigrate(&entities.SignupVerificationCode{}))
	require.NoError(t, testDB.Migrator().DropIndex(&entities.SignupVerificationCode{}, "idx_signup_verification_codes_email_unique"))

	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	codes := []entities.SignupVerificationCode{
		{
			Base: entities.Base{
				ID:        "code-old",
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
			Email:             " Alice@Example.COM ",
			CodeHash:          "old-hash",
			ExpiresAt:         baseTime.Add(time.Hour),
			ResendAvailableAt: baseTime,
		},
		{
			Base: entities.Base{
				ID:        "code-mid",
				CreatedAt: baseTime.Add(time.Minute),
				UpdatedAt: baseTime.Add(time.Minute),
			},
			Email:             "alice@example.com",
			CodeHash:          "mid-hash",
			ExpiresAt:         baseTime.Add(time.Hour),
			ResendAvailableAt: baseTime,
		},
		{
			Base: entities.Base{
				ID:        "code-new",
				CreatedAt: baseTime.Add(2 * time.Minute),
				UpdatedAt: baseTime.Add(2 * time.Minute),
			},
			Email:             "alice@example.com",
			CodeHash:          "new-hash",
			ExpiresAt:         baseTime.Add(time.Hour),
			ResendAvailableAt: baseTime,
		},
		{
			Base: entities.Base{
				ID:        "code-deleted",
				CreatedAt: baseTime.Add(3 * time.Minute),
				UpdatedAt: baseTime.Add(3 * time.Minute),
				DeletedAt: gorm.DeletedAt{Time: baseTime.Add(4 * time.Minute), Valid: true},
			},
			Email:             "ALICE@example.com",
			CodeHash:          "deleted-hash",
			ExpiresAt:         baseTime.Add(time.Hour),
			ResendAvailableAt: baseTime,
		},
	}
	for i := range codes {
		require.NoError(t, testDB.Create(&codes[i]).Error)
	}

	require.NoError(t, Migrate())

	var remaining []entities.SignupVerificationCode
	require.NoError(t, testDB.Unscoped().Where("email = ?", "alice@example.com").Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, "code-new", remaining[0].ID)
	assert.Equal(t, "new-hash", remaining[0].CodeHash)
	assert.True(t, testDB.Migrator().HasIndex(&entities.SignupVerificationCode{}, "idx_signup_verification_codes_email_unique"))

	duplicate := entities.SignupVerificationCode{
		Base:              entities.Base{ID: "code-duplicate"},
		Email:             "alice@example.com",
		CodeHash:          "duplicate-hash",
		ExpiresAt:         baseTime.Add(time.Hour),
		ResendAvailableAt: baseTime,
	}
	assert.Error(t, testDB.Create(&duplicate).Error)
}

func TestMigrateNormalizesLegacyEnvNamespacesBeforeCreatingUniqueIndex(t *testing.T) {
	testDB := useSQLiteMigrationTestDB(t)
	require.NoError(t, testDB.AutoMigrate(&entities.Env{}))
	require.NoError(t, testDB.Migrator().DropIndex(&entities.Env{}, "idx_cluster_namespace"))

	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	envs := []entities.Env{
		{
			Base:             entities.Base{ID: "env-a", CreatedAt: baseTime, UpdatedAt: baseTime, DeletedAt: gorm.DeletedAt{Time: baseTime.Add(time.Hour), Valid: true}},
			Slug:             "env-a",
			Name:             "Environment A",
			ProjectID:        "project-a",
			ClusterID:        "cluster-1",
			ClusterNamespace: " shared ",
		},
		{
			Base:             entities.Base{ID: "env-b", CreatedAt: baseTime, UpdatedAt: baseTime},
			Slug:             "env-b",
			Name:             "Environment B",
			ProjectID:        "project-b",
			ClusterID:        "cluster-1",
			ClusterNamespace: "shared",
		},
		{
			Base:             entities.Base{ID: "env-c", CreatedAt: baseTime, UpdatedAt: baseTime},
			Slug:             "env-c",
			Name:             "Environment C",
			ProjectID:        "project-c",
			ClusterID:        "cluster-1",
			ClusterNamespace: "shared",
		},
		{
			Base:             entities.Base{ID: "env-empty-a", CreatedAt: baseTime, UpdatedAt: baseTime},
			Slug:             "env-empty-a",
			Name:             "Environment Empty A",
			ProjectID:        "project-empty-a",
			ClusterID:        "cluster-1",
			ClusterNamespace: "",
		},
		{
			Base:             entities.Base{ID: "env-empty-b", CreatedAt: baseTime, UpdatedAt: baseTime},
			Slug:             "env-empty-b",
			Name:             "Environment Empty B",
			ProjectID:        "project-empty-b",
			ClusterID:        "cluster-1",
			ClusterNamespace: "   ",
		},
	}
	for i := range envs {
		require.NoError(t, testDB.Create(&envs[i]).Error)
	}

	require.NoError(t, Migrate())

	var namespaceOwner entities.Env
	require.NoError(t, testDB.First(&namespaceOwner, "id = ?", "env-b").Error)
	assert.Equal(t, "shared", namespaceOwner.ClusterNamespace)

	var clearedCount int64
	require.NoError(t, testDB.Unscoped().Model(&entities.Env{}).
		Where("id IN ? AND cluster_namespace IS NULL", []string{"env-a", "env-c", "env-empty-a", "env-empty-b"}).
		Count(&clearedCount).Error)
	assert.EqualValues(t, 4, clearedCount)
	assert.True(t, testDB.Migrator().HasIndex(&entities.Env{}, "idx_cluster_namespace"))

	duplicate := entities.Env{
		Base:             entities.Base{ID: "env-duplicate"},
		Slug:             "env-duplicate",
		Name:             "Environment Duplicate",
		ProjectID:        "project-duplicate",
		ClusterID:        "cluster-1",
		ClusterNamespace: "shared",
	}
	assert.Error(t, testDB.Create(&duplicate).Error)

	var foreignKeyCount int
	require.NoError(t, testDB.Raw("SELECT COUNT(*) FROM pragma_foreign_key_list('envs')").Scan(&foreignKeyCount).Error)
	assert.Zero(t, foreignKeyCount)
}

func TestMigrateBackfillsAndDeduplicatesLegacyBuildRepositoryNumbers(t *testing.T) {
	testDB := useSQLiteMigrationTestDB(t)
	require.NoError(t, testDB.AutoMigrate(&entities.BuildSetting{}, &entities.Build{}))
	for _, indexName := range []string{"idx_builds_code_repository_number", "idx_builds_code_repository_id"} {
		if testDB.Migrator().HasIndex(&entities.Build{}, indexName) {
			require.NoError(t, testDB.Migrator().DropIndex(&entities.Build{}, indexName))
		}
	}
	require.NoError(t, testDB.Migrator().DropColumn(&entities.Build{}, "CodeRepositoryID"))

	repositoryID := "repository-1"
	settings := []entities.BuildSetting{
		{
			ID:               "setting-a",
			Name:             "Setting A",
			CodeRepositoryID: &repositoryID,
			ImageName:        "image-a",
			RegistryID:       "registry-1",
		},
		{
			ID:               "setting-b",
			Name:             "Setting B",
			CodeRepositoryID: &repositoryID,
			ImageName:        "image-b",
			RegistryID:       "registry-1",
		},
		{
			ID:               "setting-orphan",
			Name:             "Orphan Setting",
			CodeRepositoryID: nil,
			ImageName:        "image-orphan",
			RegistryID:       "registry-1",
		},
	}
	for i := range settings {
		require.NoError(t, testDB.Create(&settings[i]).Error)
	}

	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	legacyBuilds := []struct {
		ID             string
		CreatedAt      time.Time
		BuildSettingID string
		BuildNumber    int
	}{
		{ID: "build-a", CreatedAt: baseTime, BuildSettingID: "setting-a", BuildNumber: 1},
		{ID: "build-b", CreatedAt: baseTime.Add(time.Minute), BuildSettingID: "setting-b", BuildNumber: 1},
		{ID: "build-c", CreatedAt: baseTime.Add(2 * time.Minute), BuildSettingID: "setting-a", BuildNumber: 3},
		{ID: "build-orphan", CreatedAt: baseTime.Add(3 * time.Minute), BuildSettingID: "setting-orphan", BuildNumber: 1},
	}
	for _, build := range legacyBuilds {
		require.NoError(t, testDB.Exec(`INSERT INTO builds (
			id, created_at, updated_at, build_setting_id, build_number, status, build_env_id, trigger_type
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			build.ID,
			build.CreatedAt,
			build.CreatedAt,
			build.BuildSettingID,
			build.BuildNumber,
			entities.BuildStatusSucceeded,
			"build-env-1",
			entities.BuildTriggerManual,
		).Error)
	}

	require.NoError(t, Migrate())

	var builds []entities.Build
	require.NoError(t, testDB.Order("id ASC").Find(&builds).Error)
	require.Len(t, builds, 4)
	buildsByID := make(map[string]entities.Build, len(builds))
	for _, build := range builds {
		buildsByID[build.ID] = build
	}
	for _, id := range []string{"build-a", "build-b", "build-c"} {
		require.NotNil(t, buildsByID[id].CodeRepositoryID)
		assert.Equal(t, repositoryID, *buildsByID[id].CodeRepositoryID)
	}
	assert.Equal(t, 1, buildsByID["build-a"].BuildNumber)
	assert.Equal(t, 4, buildsByID["build-b"].BuildNumber)
	assert.Equal(t, 3, buildsByID["build-c"].BuildNumber)
	assert.Nil(t, buildsByID["build-orphan"].CodeRepositoryID)
	assert.True(t, testDB.Migrator().HasIndex(&entities.Build{}, "idx_builds_code_repository_number"))

	require.NoError(t, Migrate())
	var buildBAfterSecondMigration entities.Build
	require.NoError(t, testDB.First(&buildBAfterSecondMigration, "id = ?", "build-b").Error)
	assert.Equal(t, 4, buildBAfterSecondMigration.BuildNumber)

	duplicateRepositoryID := repositoryID
	duplicate := entities.Build{
		ID:               "build-duplicate",
		BuildSettingID:   "setting-a",
		CodeRepositoryID: &duplicateRepositoryID,
		BuildNumber:      1,
		Status:           entities.BuildStatusPending,
		BuildEnvID:       "build-env-1",
		TriggerType:      entities.BuildTriggerManual,
	}
	assert.Error(t, testDB.Create(&duplicate).Error)

	var foreignKeyCount int
	require.NoError(t, testDB.Raw("SELECT COUNT(*) FROM pragma_foreign_key_list('builds')").Scan(&foreignKeyCount).Error)
	assert.Zero(t, foreignKeyCount)
}

func TestMigrateNormalizesLegacyBuildRepositoryIDsBeforeCreatingUniqueIndex(t *testing.T) {
	testDB := useSQLiteMigrationTestDB(t)
	require.NoError(t, testDB.AutoMigrate(&entities.BuildSetting{}, &entities.Build{}))
	for _, indexName := range []string{"idx_builds_code_repository_number", "idx_builds_code_repository_id"} {
		if testDB.Migrator().HasIndex(&entities.Build{}, indexName) {
			require.NoError(t, testDB.Migrator().DropIndex(&entities.Build{}, indexName))
		}
	}

	rawRepositoryID := " repo "
	trimmedRepositoryID := "repo"
	require.NoError(t, testDB.Create(&entities.BuildSetting{
		ID:               "setting-trim",
		CodeRepositoryID: &trimmedRepositoryID,
		Name:             "Trim setting",
		ImageName:        "image",
		RegistryID:       "registry",
	}).Error)

	baseTime := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	for _, build := range []entities.Build{
		{
			ID:               "build-trim-a",
			CreatedAt:        baseTime,
			UpdatedAt:        baseTime,
			BuildSettingID:   "setting-trim",
			CodeRepositoryID: &rawRepositoryID,
			BuildNumber:      1,
			Status:           entities.BuildStatusSucceeded,
			BuildEnvID:       "build-env",
			TriggerType:      entities.BuildTriggerManual,
		},
		{
			ID:               "build-trim-b",
			CreatedAt:        baseTime.Add(time.Minute),
			UpdatedAt:        baseTime.Add(time.Minute),
			BuildSettingID:   "setting-trim",
			CodeRepositoryID: &trimmedRepositoryID,
			BuildNumber:      1,
			Status:           entities.BuildStatusSucceeded,
			BuildEnvID:       "build-env",
			TriggerType:      entities.BuildTriggerManual,
		},
	} {
		require.NoError(t, testDB.Create(&build).Error)
	}

	require.NoError(t, Migrate())

	var builds []entities.Build
	require.NoError(t, testDB.Where("id IN ?", []string{"build-trim-a", "build-trim-b"}).Order("id ASC").Find(&builds).Error)
	require.Len(t, builds, 2)
	assert.Equal(t, "repo", dereferenceBuildRepositoryID(t, builds[0]))
	assert.Equal(t, "repo", dereferenceBuildRepositoryID(t, builds[1]))
	assert.Equal(t, 1, builds[0].BuildNumber)
	assert.Equal(t, 2, builds[1].BuildNumber)
}

func dereferenceBuildRepositoryID(t *testing.T, build entities.Build) string {
	t.Helper()
	if build.CodeRepositoryID == nil {
		return ""
	}
	return *build.CodeRepositoryID
}
