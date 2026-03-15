package services

import (
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
		&entities.BuildSetting{},
		&entities.Build{},
		&entities.BuildDeployment{},
		&entities.Env{},
	))

	db.DB = testDB
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
