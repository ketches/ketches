package core

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAppContextTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/app-context.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.Project{},
		&entities.Cluster{},
		&entities.Env{},
		&entities.App{},
		&entities.AppEnvVar{},
		&entities.AppVolume{},
		&entities.AppConfigFile{},
		&entities.AppProbe{},
		&entities.AppGateway{},
		&entities.AppGatewayHTTPRoute{},
		&entities.AppGatewayHTTPRouteBackend{},
		&entities.AppAutoScaling{},
		&entities.AppSchedulingRule{},
		&entities.Plugin{},
		&entities.AppPlugin{},
	))
	db.DB = testDB
}

func TestLoadAppContextLoadsExplicitRelationships(t *testing.T) {
	setupAppContextTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"}, Slug: "project", Name: "Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base: entities.Base{ID: "cluster-1"}, Slug: "cluster", Name: "Cluster", KubeConfig: "encrypted",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base: entities.Base{ID: "env-1"}, Slug: "production", Name: "Production",
		ProjectID: "project-1", ClusterID: "cluster-1", ClusterNamespace: "production",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base: entities.Base{ID: "app-1"}, Slug: "api", Name: "API", EnvID: "env-1", ContainerImage: "api:v1",
	}).Error)

	require.NoError(t, db.DB.Create([]entities.AppEnvVar{
		{ID: "env-b", AppID: "app-1", Key: "B", Value: "two"},
		{ID: "env-a", AppID: "app-1", Key: "A", Value: "one"},
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppVolume{
		ID: "volume-1", AppID: "app-1", Slug: "data", MountPath: "/data", VolumeType: "pvc",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppConfigFile{
		ID: "config-1", AppID: "app-1", Slug: "app.conf", MountPath: "/etc/app.conf", Content: "value",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppProbe{
		ID: "probe-1", AppID: "app-1", Type: "readiness", ProbeMode: "http",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGateway{
		ID: "gateway-1", AppID: "app-1", Port: 8080, Protocol: "HTTP",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGatewayHTTPRoute{
		ID: "route-1", AppGatewayID: "gateway-1", Host: "api.example.com", ListenerProtocol: "HTTP", Path: "/",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGatewayHTTPRouteBackend{
		ID: "backend-1", RouteID: "route-1", BackendAppID: "app-1", BackendPort: 8080,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppAutoScaling{
		ID: "autoscaling-1", AppID: "app-1", MinReplicas: 2, MaxReplicas: 4,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppSchedulingRule{
		ID: "scheduling-1", AppID: "app-1", RuleType: "nodeName", NodeName: "worker-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Plugin{
		ID: "plugin-1", ProjectID: "project-1", Slug: "sidecar", Name: "Sidecar", Image: "sidecar:v1", PluginType: "sidecar",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppPlugin{
		ID: "app-plugin-1", AppID: "app-1", PluginID: "plugin-1", Enabled: true,
	}).Error)

	appCtx, err := LoadAppContext(context.Background(), "app-1")
	require.NoError(t, err)
	assert.Equal(t, "api:v1", appCtx.App.ContainerImage)
	assert.Equal(t, "production", appCtx.EnvContext.Env.ClusterNamespace)
	assert.Equal(t, "Project", appCtx.EnvContext.Project.Name)
	assert.Equal(t, "Cluster", appCtx.EnvContext.Cluster.Name)
	require.Len(t, appCtx.EnvVars, 2)
	assert.Equal(t, []string{"env-a", "env-b"}, []string{appCtx.EnvVars[0].ID, appCtx.EnvVars[1].ID})
	require.Len(t, appCtx.Volumes, 1)
	require.Len(t, appCtx.ConfigFiles, 1)
	require.Len(t, appCtx.Probes, 1)
	require.Len(t, appCtx.Gateways, 1)
	require.Len(t, appCtx.GatewayRoutes, 1)
	require.Len(t, appCtx.GatewayBackends, 1)
	require.NotNil(t, appCtx.AutoScaling)
	assert.Equal(t, 4, appCtx.AutoScaling.MaxReplicas)
	require.NotNil(t, appCtx.SchedulingRule)
	assert.Equal(t, "worker-1", appCtx.SchedulingRule.NodeName)
	require.Len(t, appCtx.AppPlugins, 1)
	require.Contains(t, appCtx.Plugins, "plugin-1")
	assert.Equal(t, "sidecar:v1", appCtx.Plugins["plugin-1"].Image)
}
