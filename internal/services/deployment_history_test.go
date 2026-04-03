package services

import (
	"context"
	"testing"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDeploymentHistoryTestDB(t *testing.T) {
	t.Helper()

	setupAppVolumeTestDB(t)
	require.NoError(t, db.DB.AutoMigrate(&entities.DeploymentHistory{}))
}

func seedDeploymentHistoryApp(t *testing.T, image string) {
	t.Helper()

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo-project",
		Name: "Demo Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "demo-cluster",
		Name:       "Demo Cluster",
		KubeConfig: "test",
		Enabled:    true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "prod",
		Name:             "Production",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "work",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-1"},
		Slug:           "api",
		Name:           "API",
		EnvID:          "env-1",
		AppType:        "Deployment",
		ContainerImage: image,
		Replicas:       2,
		RequestCPU:     250,
		RequestMemory:  256,
		LimitCPU:       500,
		LimitMemory:    512,
	}).Error)
}

func TestUpdateAppImageRecordsDeploymentHistoryOnImageChange(t *testing.T) {
	setupDeploymentHistoryTestDB(t)
	seedDeploymentHistoryApp(t, "nginx:1.25")

	originalApplyAppFn := applyAppFn
	applyAppFn = func(_ context.Context, _ *models.AppContext) error {
		return nil
	}
	t.Cleanup(func() {
		applyAppFn = originalApplyAppFn
	})

	appCtx, err := UpdateAppImage(context.Background(), "app-1", &models.UpdateAppImageRequest{
		ContainerImage: "nginx:1.26",
	})

	require.NoError(t, err)
	require.NotNil(t, appCtx)
	assert.Equal(t, "nginx:1.26", appCtx.App.ContainerImage)

	var histories []entities.DeploymentHistory
	require.NoError(t, db.DB.Order("created_at ASC").Find(&histories).Error)
	require.Len(t, histories, 1)
	assert.Equal(t, "app-1", histories[0].AppID)
	assert.Equal(t, "nginx:1.25", histories[0].ImageBefore)
	assert.Equal(t, "nginx:1.26", histories[0].ImageAfter)
}

func TestUpdateAppImageSkipsDeploymentHistoryWhenImageUnchanged(t *testing.T) {
	setupDeploymentHistoryTestDB(t)
	seedDeploymentHistoryApp(t, "nginx:1.25")

	originalApplyAppFn := applyAppFn
	applyAppFn = func(_ context.Context, _ *models.AppContext) error {
		return nil
	}
	t.Cleanup(func() {
		applyAppFn = originalApplyAppFn
	})

	appCtx, err := UpdateAppImage(context.Background(), "app-1", &models.UpdateAppImageRequest{
		ContainerImage: "nginx:1.25",
	})

	require.NoError(t, err)
	require.NotNil(t, appCtx)

	var historyCount int64
	require.NoError(t, db.DB.Model(&entities.DeploymentHistory{}).Count(&historyCount).Error)
	assert.EqualValues(t, 0, historyCount)
}

func TestRollbackDeploymentRestoresImageOnly(t *testing.T) {
	setupDeploymentHistoryTestDB(t)
	seedDeploymentHistoryApp(t, "nginx:1.26")

	require.NoError(t, db.DB.Create(&entities.DeploymentHistory{
		ID:                  "history-1",
		AppID:               "app-1",
		ImageBefore:         "nginx:1.25",
		ImageAfter:          "nginx:1.26",
		ReplicasBefore:      1,
		ReplicasAfter:       2,
		RequestCPUBefore:    100,
		RequestCPUAfter:     250,
		RequestMemoryBefore: 128,
		RequestMemoryAfter:  256,
		LimitCPUBefore:      200,
		LimitCPUAfter:       500,
		LimitMemoryBefore:   256,
		LimitMemoryAfter:    512,
		DeployType:          "manual",
		DeployedBy:          "user-1",
		Reason:              "Container image updated",
		Status:              "success",
	}).Error)

	originalApplyAppContextFn := applyAppContextFn
	applyAppContextFn = func(_ context.Context, _ *models.AppContext) error {
		return nil
	}
	t.Cleanup(func() {
		applyAppContextFn = originalApplyAppContextFn
	})

	app, err := RollbackDeployment("app-1", "history-1")

	require.NoError(t, err)
	require.NotNil(t, app)
	assert.Equal(t, "nginx:1.25", app.ContainerImage)
	assert.Equal(t, 2, app.Replicas)
	assert.Equal(t, 250, app.RequestCPU)
	assert.Equal(t, 256, app.RequestMemory)
	assert.Equal(t, 500, app.LimitCPU)
	assert.Equal(t, 512, app.LimitMemory)

	var persisted entities.App
	require.NoError(t, db.DB.First(&persisted, "id = ?", "app-1").Error)
	assert.Equal(t, "nginx:1.25", persisted.ContainerImage)
	assert.Equal(t, 2, persisted.Replicas)
	assert.Equal(t, 250, persisted.RequestCPU)
	assert.Equal(t, 256, persisted.RequestMemory)
	assert.Equal(t, 500, persisted.LimitCPU)
	assert.Equal(t, 512, persisted.LimitMemory)

	var histories []entities.DeploymentHistory
	require.NoError(t, db.DB.Order("created_at ASC").Find(&histories).Error)
	require.Len(t, histories, 2)
	assert.Equal(t, "rollback", histories[1].DeployType)
	assert.Equal(t, "nginx:1.26", histories[1].ImageBefore)
	assert.Equal(t, "nginx:1.25", histories[1].ImageAfter)
	assert.Equal(t, "Rollback to previous image version only", histories[1].Reason)
	assert.Equal(t, 2, histories[1].ReplicasBefore)
	assert.Equal(t, 2, histories[1].ReplicasAfter)
}

func TestExecuteRollbackActionRequiresDeploymentHistoryTarget(t *testing.T) {
	appCtx := &models.AppContext{
		App: entities.App{
			Base: entities.Base{ID: "app-1"},
			Slug: "api",
		},
	}

	result, err := executeRollbackAction(context.Background(), appCtx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "rollback requires a deployment history target", err.Error())
}
