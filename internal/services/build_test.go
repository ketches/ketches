package services

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestToBuildResponse_ExposesLogPersistMetadata(t *testing.T) {
	persistedAt := time.Date(2026, 3, 15, 12, 5, 0, 0, time.UTC)
	expireAt := persistedAt.Add(15 * 24 * time.Hour)
	build := &entities.Build{
		ID:               "build-1",
		BuildSettingID:   "setting-1",
		BuildEnvID:       "env-1",
		BuildNumber:      1,
		Status:           entities.BuildStatusFailed,
		TriggerType:      entities.BuildTriggerManual,
		LogPersistStatus: entities.BuildLogPersistFailed,
		LogPersistError:  "archive failed",
		LogPersistedAt:   &persistedAt,
		LogExpireAt:      &expireAt,
	}

	resp := ToBuildResponse(context.Background(), build)

	assert.Equal(t, string(entities.BuildLogPersistFailed), resp.LogPersistStatus)
	assert.Equal(t, "archive failed", resp.LogPersistError)
}

func setupBuildServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.App{},
		&entities.CodeRepository{},
		&entities.ContainerRegistry{},
		&entities.BuildSetting{},
		&entities.Build{},
		&entities.BuildDeployment{},
		&entities.Env{},
		&entities.Project{},
	))

	db.DB = testDB
}

func TestValidateCodeRepositoryAutoDeployTargetRequiresProjectScopedEnvironment(t *testing.T) {
	setupBuildServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "deploy-env"},
		Slug:      "deploy",
		Name:      "Deploy",
		ProjectID: "project-other",
		ClusterID: "cluster-1",
	}).Error)

	req := &models.TriggerCodeRepositoryBuildRequest{DeployEnvID: "deploy-env"}
	err := validateCodeRepositoryAutoDeployTarget("project-repo", req)

	require.Error(t, err)
	assert.ErrorContains(t, err, "same project as the code repository")
}

func TestValidateCodeRepositoryAutoDeployTargetRequiresAppInDeployEnvironment(t *testing.T) {
	setupBuildServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "deploy-env"},
		Slug:      "deploy",
		Name:      "Deploy",
		ProjectID: "project-repo",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "other-env"},
		Slug:      "other",
		Name:      "Other",
		ProjectID: "project-repo",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-1"},
		Slug:           "app-1",
		Name:           "App 1",
		EnvID:          "other-env",
		ContainerImage: "nginx:1",
	}).Error)

	req := &models.TriggerCodeRepositoryBuildRequest{
		DeployEnvID: "deploy-env",
		DeployAppID: "app-1",
	}
	err := validateCodeRepositoryAutoDeployTarget("project-repo", req)

	require.Error(t, err)
	assert.ErrorContains(t, err, "deploy environment")
}

func TestTriggerCodeRepositoryBuildRejectsCrossProjectAutoDeployBeforeCreatingRecords(t *testing.T) {
	setupBuildServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-repo"},
		Slug: "repo-project",
		Name: "Repo Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-other"},
		Slug: "other-project",
		Name: "Other Project",
	}).Error)
	repo := entities.CodeRepository{
		Base:       entities.Base{ID: "repo-1"},
		ProjectID:  "project-repo",
		Name:       "Repo",
		Slug:       "repo",
		GitRepoURL: "https://example.com/repo.git",
	}
	require.NoError(t, db.DB.Create(&repo).Error)
	require.NoError(t, db.DB.Create(&entities.ContainerRegistry{
		ID:        "registry-1",
		Name:      "Registry",
		Provider:  entities.RegistryProviderCustom,
		Endpoint:  "registry.example.com",
		Scope:     entities.RegistryScopeProject,
		ProjectID: "project-repo",
		Enabled:   true,
	}).Error)
	repoID := repo.ID
	require.NoError(t, db.DB.Create(&entities.BuildSetting{
		ID:               "setting-1",
		CodeRepositoryID: &repoID,
		Name:             "Build",
		ImageName:        "repo/app",
		RegistryID:       "registry-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:       entities.Base{ID: "build-env"},
		Slug:       "build",
		Name:       "Build",
		ProjectID:  "project-repo",
		ClusterID:  "cluster-1",
		IsBuildEnv: true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "deploy-env"},
		Slug:      "deploy",
		Name:      "Deploy",
		ProjectID: "project-other",
		ClusterID: "cluster-1",
	}).Error)
	autoDeploy := true

	build, err := TriggerCodeRepositoryBuild("repo-1", "user-1", &models.TriggerCodeRepositoryBuildRequest{
		BuildSettingID: "setting-1",
		BuildEnvID:     "build-env",
		AutoDeploy:     &autoDeploy,
		DeployEnvID:    "deploy-env",
	})

	require.Error(t, err)
	assert.Nil(t, build)
	var buildCount, deploymentCount int64
	require.NoError(t, db.DB.Model(&entities.Build{}).Count(&buildCount).Error)
	require.NoError(t, db.DB.Model(&entities.BuildDeployment{}).Count(&deploymentCount).Error)
	assert.Zero(t, buildCount)
	assert.Zero(t, deploymentCount)
}

func TestBuildRepositoryNumberCompositeUniqueIndex(t *testing.T) {
	setupBuildServiceTestDB(t)

	repositoryID := "repo-1"
	otherRepositoryID := "repo-2"
	require.NoError(t, db.DB.Create(&entities.Build{
		ID:               "build-1",
		BuildSettingID:   "setting-1",
		CodeRepositoryID: &repositoryID,
		BuildEnvID:       "build-env",
		BuildNumber:      1,
		Status:           entities.BuildStatusSucceeded,
	}).Error)

	err := db.DB.Create(&entities.Build{
		ID:               "build-duplicate",
		BuildSettingID:   "setting-2",
		CodeRepositoryID: &repositoryID,
		BuildEnvID:       "build-env",
		BuildNumber:      1,
		Status:           entities.BuildStatusSucceeded,
	}).Error
	assert.Error(t, err)
	require.NoError(t, db.DB.Create(&entities.Build{
		ID:               "build-other-repository",
		BuildSettingID:   "setting-3",
		CodeRepositoryID: &otherRepositoryID,
		BuildEnvID:       "build-env",
		BuildNumber:      1,
		Status:           entities.BuildStatusSucceeded,
	}).Error)
}

func TestNextCodeRepositoryBuildNumberUsesRepositoryScope(t *testing.T) {
	setupBuildServiceTestDB(t)

	repositoryID := "repo-1"
	otherRepositoryID := "repo-2"
	settings := []entities.BuildSetting{
		{ID: "setting-1", CodeRepositoryID: &repositoryID, ImageName: "repo/app", RegistryID: "registry-1"},
		{ID: "setting-2", CodeRepositoryID: &repositoryID, ImageName: "repo/worker", RegistryID: "registry-1"},
		{ID: "setting-3", CodeRepositoryID: &otherRepositoryID, ImageName: "other/app", RegistryID: "registry-1"},
	}
	require.NoError(t, db.DB.Create(&settings).Error)
	builds := []entities.Build{
		{ID: "build-1", BuildSettingID: "setting-1", CodeRepositoryID: &repositoryID, BuildEnvID: "env-1", BuildNumber: 1, Status: entities.BuildStatusSucceeded},
		{ID: "build-2", BuildSettingID: "setting-2", CodeRepositoryID: &repositoryID, BuildEnvID: "env-1", BuildNumber: 3, Status: entities.BuildStatusSucceeded},
		{ID: "build-3", BuildSettingID: "setting-3", CodeRepositoryID: &otherRepositoryID, BuildEnvID: "env-1", BuildNumber: 7, Status: entities.BuildStatusSucceeded},
	}
	require.NoError(t, db.DB.Create(&builds).Error)

	next, err := nextCodeRepositoryBuildNumber(db.DB, repositoryID)
	require.NoError(t, err)
	assert.Equal(t, 4, next)
	otherNext, err := nextCodeRepositoryBuildNumber(db.DB, otherRepositoryID)
	require.NoError(t, err)
	assert.Equal(t, 8, otherNext)
}

func TestNextCodeRepositoryBuildNumberContinuesAfterBuildSettingDeletion(t *testing.T) {
	setupBuildServiceTestDB(t)

	repositoryID := "repo-1"
	oldSetting := entities.BuildSetting{
		ID:               "setting-old",
		CodeRepositoryID: &repositoryID,
		Name:             "Old",
		ImageName:        "repo/old",
		RegistryID:       "registry-1",
	}
	require.NoError(t, db.DB.Create(&oldSetting).Error)
	require.NoError(t, db.DB.Create(&entities.Build{
		ID:               "build-history",
		BuildSettingID:   oldSetting.ID,
		CodeRepositoryID: &repositoryID,
		BuildEnvID:       "env-1",
		BuildNumber:      7,
		Status:           entities.BuildStatusSucceeded,
	}).Error)

	require.NoError(t, DeleteRepoBuildSetting(oldSetting.ID))
	newSetting := entities.BuildSetting{
		ID:               "setting-new",
		CodeRepositoryID: &repositoryID,
		Name:             "New",
		ImageName:        "repo/new",
		RegistryID:       "registry-1",
	}
	require.NoError(t, db.DB.Create(&newSetting).Error)

	next, err := nextCodeRepositoryBuildNumber(db.DB, repositoryID)
	require.NoError(t, err)
	assert.Equal(t, 8, next)
	require.NoError(t, db.DB.Create(&entities.Build{
		ID:               "build-new",
		BuildSettingID:   newSetting.ID,
		CodeRepositoryID: &repositoryID,
		BuildEnvID:       "env-1",
		BuildNumber:      next,
		Status:           entities.BuildStatusPending,
	}).Error)

	var buildCount int64
	require.NoError(t, db.DB.Model(&entities.Build{}).Where("code_repository_id = ?", repositoryID).Count(&buildCount).Error)
	assert.EqualValues(t, 2, buildCount)
}

func TestListDeployedAppsByEnvironmentAndBuildSetting(t *testing.T) {
	setupBuildServiceTestDB(t)

	now := time.Now()
	envID := "env-1"
	otherEnvID := "env-2"

	app1 := entities.App{Base: entities.Base{ID: "app-1", CreatedAt: now.Add(-2 * time.Minute)}, Name: "App One", EnvID: envID, Slug: "app-one", AppType: "Deployment", ContainerImage: "nginx:1", Replicas: 1}
	app2 := entities.App{Base: entities.Base{ID: "app-2", CreatedAt: now.Add(-1 * time.Minute)}, Name: "App Two", EnvID: envID, Slug: "app-two", AppType: "Deployment", ContainerImage: "nginx:1", Replicas: 1}
	app3 := entities.App{Base: entities.Base{ID: "app-3", CreatedAt: now}, Name: "Other Repo App", EnvID: envID, Slug: "other-repo-app", AppType: "Deployment", ContainerImage: "nginx:1", Replicas: 1}
	require.NoError(t, db.DB.Create(&app1).Error)
	require.NoError(t, db.DB.Create(&app2).Error)
	require.NoError(t, db.DB.Create(&app3).Error)

	buildSetting1 := entities.BuildSetting{ID: "bs-1", ImageName: "repo/app-1", RegistryID: "reg-1", Name: "frontend"}
	buildSetting2 := entities.BuildSetting{ID: "bs-2", ImageName: "repo/app-2", RegistryID: "reg-1", Name: "backend"}
	require.NoError(t, db.DB.Create(&buildSetting1).Error)
	require.NoError(t, db.DB.Create(&buildSetting2).Error)

	build1 := entities.Build{ID: "build-1", BuildSettingID: buildSetting1.ID, BuildEnvID: "build-env", BuildNumber: 1, Status: entities.BuildStatusSucceeded}
	build2 := entities.Build{ID: "build-2", BuildSettingID: buildSetting1.ID, BuildEnvID: "build-env", BuildNumber: 2, Status: entities.BuildStatusSucceeded}
	build3 := entities.Build{ID: "build-3", BuildSettingID: buildSetting2.ID, BuildEnvID: "build-env", BuildNumber: 3, Status: entities.BuildStatusSucceeded}
	require.NoError(t, db.DB.Create(&build1).Error)
	require.NoError(t, db.DB.Create(&build2).Error)
	require.NoError(t, db.DB.Create(&build3).Error)

	app1ID := app1.ID
	app2ID := app2.ID
	app3ID := app3.ID
	require.NoError(t, db.DB.Create(&entities.BuildDeployment{ID: "bd-1", BuildID: build1.ID, AppID: &app1ID, EnvID: envID, Status: entities.BuildDeploymentStatusDeployed, DeployedBy: "user-1"}).Error)
	require.NoError(t, db.DB.Create(&entities.BuildDeployment{ID: "bd-2", BuildID: build2.ID, AppID: &app1ID, EnvID: envID, Status: entities.BuildDeploymentStatusDeployed, DeployedBy: "user-1"}).Error)
	require.NoError(t, db.DB.Create(&entities.BuildDeployment{ID: "bd-3", BuildID: build2.ID, AppID: &app2ID, EnvID: envID, Status: entities.BuildDeploymentStatusDeployed, DeployedBy: "user-1"}).Error)
	require.NoError(t, db.DB.Create(&entities.BuildDeployment{ID: "bd-4", BuildID: build1.ID, AppID: &app2ID, EnvID: otherEnvID, Status: entities.BuildDeploymentStatusDeployed, DeployedBy: "user-1"}).Error)
	require.NoError(t, db.DB.Create(&entities.BuildDeployment{ID: "bd-5", BuildID: build3.ID, AppID: &app3ID, EnvID: envID, Status: entities.BuildDeploymentStatusDeployed, DeployedBy: "user-1"}).Error)
	require.NoError(t, db.DB.Create(&entities.BuildDeployment{ID: "bd-6", BuildID: build1.ID, AppID: nil, EnvID: envID, Status: entities.BuildDeploymentStatusPending, DeployedBy: "user-1"}).Error)

	apps, err := ListDeployedAppsByEnvironmentAndBuildSetting(envID, buildSetting1.ID)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	assert.Equal(t, "app-2", apps[0].ID)
	assert.Equal(t, "App Two", apps[0].Name)
	assert.Equal(t, "app-1", apps[1].ID)
	assert.Equal(t, "App One", apps[1].Name)
}

func TestToCodeRepositoryDeploymentResponse_AllowsNilAppID(t *testing.T) {
	resp := toCodeRepositoryDeploymentResponse(&codeRepositoryDeploymentRow{
		BuildID:                "build-1",
		BuildSettingID:         "bs-1",
		BuildNumber:            3,
		GitRef:                 "main",
		ImageFullName:          "ghcr.io/demo/app:v1.0.0",
		DeploymentID:           "bd-1",
		DeploymentStatus:       string(entities.BuildDeploymentStatusPending),
		DeploymentErrorMessage: "",
		AppID:                  nil,
		AppName:                "New App",
		EnvName:                "Dev",
	})

	assert.Equal(t, "", resp.AppID)
	assert.Equal(t, "New App", resp.AppName)
}

func TestListDeployments_AllowsNilAppID(t *testing.T) {
	setupBuildServiceTestDB(t)

	repoID := "repo-1"
	envID := "env-1"

	env := entities.Env{
		Base:             entities.Base{ID: envID},
		Name:             "Dev",
		Slug:             "dev",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "project-dev",
	}
	require.NoError(t, db.DB.Create(&env).Error)

	buildSetting := entities.BuildSetting{
		ID:               "bs-1",
		CodeRepositoryID: ptr("repo-1"),
		Name:             "backend",
		ImageName:        "repo/app",
		RegistryID:       "reg-1",
	}
	require.NoError(t, db.DB.Create(&buildSetting).Error)

	build := entities.Build{
		ID:             "build-1",
		BuildSettingID: buildSetting.ID,
		BuildEnvID:     envID,
		BuildNumber:    1,
		Status:         entities.BuildStatusSucceeded,
		GitRef:         "main",
		ImageFullName:  "ghcr.io/demo/app:v1.0.0",
	}
	require.NoError(t, db.DB.Create(&build).Error)

	deployment := entities.BuildDeployment{
		ID:         "bd-1",
		BuildID:    build.ID,
		AppID:      nil,
		EnvID:      envID,
		AppName:    "New App",
		Status:     entities.BuildDeploymentStatusPending,
		DeployedBy: "user-1",
	}
	require.NoError(t, db.DB.Create(&deployment).Error)

	res, err := ListDeployments(repoID)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "", res[0].AppID)
	assert.Equal(t, "New App", res[0].AppName)
	assert.Equal(t, "Dev", res[0].EnvName)
}

func TestListDeployments_ExcludesDeletedDetachedAndMissingApps(t *testing.T) {
	setupBuildServiceTestDB(t)

	repoID := "repo-1"
	envID := "env-1"

	env := entities.Env{
		Base:             entities.Base{ID: envID},
		Name:             "Dev",
		Slug:             "dev",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "project-dev",
	}
	require.NoError(t, db.DB.Create(&env).Error)

	buildSetting := entities.BuildSetting{
		ID:               "bs-1",
		CodeRepositoryID: ptr(repoID),
		Name:             "backend",
		ImageName:        "repo/app",
		RegistryID:       "reg-1",
	}
	require.NoError(t, db.DB.Create(&buildSetting).Error)

	builds := []entities.Build{
		{ID: "build-keep", BuildSettingID: buildSetting.ID, BuildEnvID: envID, BuildNumber: 1, Status: entities.BuildStatusSucceeded, GitRef: "main", ImageFullName: "ghcr.io/demo/app:v1.0.0"},
		{ID: "build-new-app", BuildSettingID: buildSetting.ID, BuildEnvID: envID, BuildNumber: 2, Status: entities.BuildStatusSucceeded, GitRef: "main", ImageFullName: "ghcr.io/demo/app:v1.0.1"},
		{ID: "build-deleted", BuildSettingID: buildSetting.ID, BuildEnvID: envID, BuildNumber: 3, Status: entities.BuildStatusSucceeded, GitRef: "main", ImageFullName: "ghcr.io/demo/app:v1.0.2"},
		{ID: "build-detached", BuildSettingID: buildSetting.ID, BuildEnvID: envID, BuildNumber: 4, Status: entities.BuildStatusSucceeded, GitRef: "main", ImageFullName: "ghcr.io/demo/app:v1.0.3"},
		{ID: "build-missing", BuildSettingID: buildSetting.ID, BuildEnvID: envID, BuildNumber: 5, Status: entities.BuildStatusSucceeded, GitRef: "main", ImageFullName: "ghcr.io/demo/app:v1.0.4"},
	}
	require.NoError(t, db.DB.Create(&builds).Error)

	keepRepoID := repoID
	keepApp := entities.App{
		Base:             entities.Base{ID: "app-keep"},
		Slug:             "app-keep",
		Name:             "Keep App",
		EnvID:            envID,
		AppType:          "Deployment",
		ContainerImage:   "nginx:1",
		Replicas:         1,
		CodeRepositoryID: &keepRepoID,
	}
	deletedRepoID := repoID
	deletedApp := entities.App{
		Base:             entities.Base{ID: "app-deleted"},
		Slug:             "app-deleted",
		Name:             "Deleted App",
		EnvID:            envID,
		AppType:          "Deployment",
		ContainerImage:   "nginx:1",
		Replicas:         1,
		CodeRepositoryID: &deletedRepoID,
	}
	detachedApp := entities.App{
		Base:           entities.Base{ID: "app-detached"},
		Slug:           "app-detached",
		Name:           "Detached App",
		EnvID:          envID,
		AppType:        "Deployment",
		ContainerImage: "nginx:1",
		Replicas:       1,
	}
	require.NoError(t, db.DB.Create(&keepApp).Error)
	require.NoError(t, db.DB.Create(&deletedApp).Error)
	require.NoError(t, db.DB.Create(&detachedApp).Error)
	require.NoError(t, db.DB.Delete(&deletedApp).Error)

	keepAppID := keepApp.ID
	deletedAppID := deletedApp.ID
	detachedAppID := detachedApp.ID
	missingAppID := "app-missing"
	deployments := []entities.BuildDeployment{
		{ID: "bd-keep", BuildID: "build-keep", AppID: &keepAppID, EnvID: envID, Status: entities.BuildDeploymentStatusDeployed, DeployedBy: "user-1"},
		{ID: "bd-new-app", BuildID: "build-new-app", AppID: nil, AppName: "New App", EnvID: envID, Status: entities.BuildDeploymentStatusPending, DeployedBy: "user-1"},
		{ID: "bd-deleted", BuildID: "build-deleted", AppID: &deletedAppID, EnvID: envID, Status: entities.BuildDeploymentStatusDeployed, DeployedBy: "user-1"},
		{ID: "bd-detached", BuildID: "build-detached", AppID: &detachedAppID, EnvID: envID, Status: entities.BuildDeploymentStatusDeployed, DeployedBy: "user-1"},
		{ID: "bd-missing", BuildID: "build-missing", AppID: &missingAppID, AppName: "Missing App", EnvID: envID, Status: entities.BuildDeploymentStatusFailed, DeployedBy: "user-1"},
	}
	require.NoError(t, db.DB.Create(&deployments).Error)

	res, err := ListDeployments(repoID)
	require.NoError(t, err)
	require.Len(t, res, 2)

	deploymentIDs := []string{res[0].DeploymentID, res[1].DeploymentID}
	assert.ElementsMatch(t, []string{"bd-keep", "bd-new-app"}, deploymentIDs)

	for _, item := range res {
		switch item.DeploymentID {
		case "bd-keep":
			assert.Equal(t, "app-keep", item.AppID)
			assert.Equal(t, "Keep App", item.AppName)
		case "bd-new-app":
			assert.Equal(t, "", item.AppID)
			assert.Equal(t, "New App", item.AppName)
		}
	}
}
