package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPermanentlyDeleteCodeRepositoryFindsBuildsAfterSettingDeletion(t *testing.T) {
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/code-repository-deletion.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.Project{},
		&entities.CodeRepository{},
		&entities.BuildSetting{},
		&entities.Build{},
		&entities.BuildDeployment{},
		&entities.DeploymentHistory{},
		&entities.App{},
	))
	db.DB = testDB

	repositoryID := "repo-1"
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.CodeRepository{
		Base:       entities.Base{ID: repositoryID},
		ProjectID:  "project-1",
		Name:       "Repository 1",
		Slug:       "repository-1",
		GitRepoURL: "https://example.com/repository-1.git",
	}).Error)

	directSettingID := "setting-direct"
	legacySettingID := "setting-legacy"
	for _, settingID := range []string{directSettingID, legacySettingID} {
		require.NoError(t, db.DB.Create(&entities.BuildSetting{
			ID:               settingID,
			CodeRepositoryID: &repositoryID,
			Name:             settingID,
			ImageName:        "example/image",
			RegistryID:       "registry-1",
		}).Error)
	}

	require.NoError(t, db.DB.Create(&entities.Build{
		ID:               "build-direct",
		BuildSettingID:   directSettingID,
		CodeRepositoryID: &repositoryID,
		BuildNumber:      1,
		Status:           entities.BuildStatusSucceeded,
		BuildEnvID:       "env-1",
		TriggerType:      entities.BuildTriggerManual,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Build{
		ID:             "build-legacy",
		BuildSettingID: legacySettingID,
		BuildNumber:    2,
		Status:         entities.BuildStatusSucceeded,
		BuildEnvID:     "env-1",
		TriggerType:    entities.BuildTriggerManual,
	}).Error)
	for _, buildID := range []string{"build-direct", "build-legacy"} {
		require.NoError(t, db.DB.Create(&entities.BuildDeployment{
			ID:      "deployment-" + buildID,
			BuildID: buildID,
			EnvID:   "env-1",
		}).Error)
	}

	require.NoError(t, db.DB.Create(&entities.App{
		Base:             entities.Base{ID: "app-1"},
		Slug:             "app-1",
		Name:             "App 1",
		EnvID:            "env-1",
		ContainerImage:   "nginx:latest",
		CodeRepositoryID: &repositoryID,
	}).Error)
	directBuildID := "build-direct"
	require.NoError(t, db.DB.Create(&entities.DeploymentHistory{
		ID:      "history-1",
		AppID:   "app-1",
		BuildID: &directBuildID,
	}).Error)

	require.NoError(t, DeleteRepoBuildSetting(directSettingID))
	require.NoError(t, db.DB.Delete(&entities.CodeRepository{}, "id = ?", repositoryID).Error)
	require.NoError(t, PermanentlyDeleteCodeRepository(repositoryID, RecycleBinActor{Role: app.UserRoleAdmin}))

	for _, model := range []any{
		&entities.CodeRepository{},
		&entities.BuildSetting{},
		&entities.Build{},
		&entities.BuildDeployment{},
	} {
		var count int64
		require.NoError(t, db.DB.Unscoped().Model(model).Count(&count).Error)
		require.Zero(t, count)
	}

	var history entities.DeploymentHistory
	require.NoError(t, db.DB.First(&history, "id = ?", "history-1").Error)
	require.Nil(t, history.BuildID)
	var application entities.App
	require.NoError(t, db.DB.First(&application, "id = ?", "app-1").Error)
	require.Nil(t, application.CodeRepositoryID)
}
