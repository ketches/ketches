package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/core/exporter"
	"github.com/ketches/ketches/internal/core/importer"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertAppsToMetadata(t *testing.T) {
	now := time.Now()
	appCtxs := []*models.AppContext{
		{
			App: entities.App{
				Name:             "test-app",
				Slug:             "test-app",
				AppType:          "Deployment",
				Description:      "Test App",
				ContainerImage:   "nginx:latest",
				ContainerCommand: "nginx",
				Replicas:         2,
				RequestCPU:       100,
				RequestMemory:    128,
				LimitCPU:         200,
				LimitMemory:      256,
				RegistryUsername: "user",
				RegistryPassword: "password",
				Base:             entities.Base{CreatedAt: now},
			},
			EnvVars: []entities.AppEnvVar{
				{Key: "ENV_KEY", Value: "ENV_VALUE"},
			},
			Gateways: []entities.AppGateway{
				{ID: "gateway-1", Port: 80, Protocol: "http", GatewayPort: 8080, ServiceType: "ClusterIP"},
			},
			GatewayRoutes: []entities.AppGatewayHTTPRoute{
				{
					ID:               "route-1",
					AppGatewayID:     "gateway-1",
					Host:             "example.com",
					ListenerProtocol: "https",
					Path:             "/",
					PathMatchType:    "PathPrefix",
					Enabled:          true,
					CertID:           strPtr("cert-id"),
				},
			},
			GatewayBackends: []entities.AppGatewayHTTPRouteBackend{
				{ID: "backend-1", RouteID: "route-1", BackendAppID: "", BackendPort: 80, Weight: 100},
			},
			ConfigFiles: []entities.AppConfigFile{
				{Slug: "config", MountPath: "/config", Content: "data", FileMode: "0644"},
			},
			Volumes: []entities.AppVolume{
				{Slug: "vol", MountPath: "/data", SubPath: "", VolumeType: "pvc", Capacity: 10, StorageClass: "standard"},
			},
			Probes: []entities.AppProbe{
				{Type: "liveness", ProbeMode: "http", Enabled: true, HttpGetPath: "/health", HttpGetPort: 80, InitialDelaySeconds: 10},
			},
			AutoScaling: &entities.AppAutoScaling{
				MinReplicas: 1, MaxReplicas: 5, TargetCPUUtilization: 80, TargetMemoryUtilization: 80,
			},
			SchedulingRule: &entities.AppSchedulingRule{
				RuleType: "nodeSelector", NodeSelector: "disktype=ssd",
			},
		},
	}

	metadatas := convertAppContextsToMetadata(appCtxs)

	assert.Len(t, metadatas, 1)
	meta := metadatas[0]

	assert.Equal(t, "test-app", meta.AppName)
	assert.Equal(t, "test-app", meta.AppSlug)
	assert.Equal(t, "Deployment", meta.AppType)
	assert.Equal(t, "Test App", meta.Description)
	assert.Equal(t, "nginx:latest", meta.ContainerImage)
	assert.Equal(t, "nginx", meta.ContainerCommand)
	assert.Equal(t, 2, meta.Replicas)
	assert.Equal(t, 100, meta.RequestCPU)
	assert.Equal(t, 128, meta.RequestMemory)
	assert.Equal(t, 200, meta.LimitCPU)
	assert.Equal(t, 256, meta.LimitMemory)
	assert.Equal(t, "user", meta.RegistryUsername)
	assert.Empty(t, meta.RegistryPassword)

	metadataJSON, err := json.Marshal(meta)
	require.NoError(t, err)
	assert.NotContains(t, string(metadataJSON), "registry_password")

	assert.Len(t, meta.EnvVars, 1)
	assert.Equal(t, "ENV_KEY", meta.EnvVars[0].Key)
	assert.Equal(t, "ENV_VALUE", meta.EnvVars[0].Value)

	assert.Len(t, meta.Gateways, 1)
	assert.Equal(t, 80, meta.Gateways[0].Port)
	assert.Equal(t, "http", meta.Gateways[0].Protocol)
	assert.Equal(t, 8080, meta.Gateways[0].GatewayPort)
	require.Len(t, meta.Gateways[0].Routes, 1)
	assert.Equal(t, true, meta.Gateways[0].Routes[0].Enabled)
	assert.Equal(t, "example.com", meta.Gateways[0].Routes[0].Host)
	assert.Equal(t, "https", meta.Gateways[0].Routes[0].ListenerProtocol)

	assert.Len(t, meta.ConfigFiles, 1)
	assert.Equal(t, "config", meta.ConfigFiles[0].Slug)
	assert.Equal(t, "0644", meta.ConfigFiles[0].FileMode)

	assert.Len(t, meta.Volumes, 1)
	assert.Equal(t, "vol", meta.Volumes[0].Slug)

	assert.Len(t, meta.Probes, 1)
	assert.Equal(t, "liveness", meta.Probes[0].Type)

	assert.NotNil(t, meta.AutoScaling)
	assert.Equal(t, 1, meta.AutoScaling.MinReplicas)

	assert.NotNil(t, meta.SchedulingRule)
	assert.Equal(t, "nodeSelector", meta.SchedulingRule.RuleType)
}

func strPtr(v string) *string { return &v }

func TestImportResultStruct(t *testing.T) {
	// Verify ImportResult struct exists and has correct fields
	res := ImportResult{}
	res.Imported = []ImportAppResult{{Name: "test"}}
	res.Conflicts = []ConflictInfo{{ExistingApp: &entities.App{}}}
	assert.NotEmpty(t, res.Imported)
	assert.NotEmpty(t, res.Conflicts)
}

func TestConflictInfoStruct(t *testing.T) {
	// Verify ConflictInfo struct
	ci := ConflictInfo{
		ExistingApp: &entities.App{Slug: "old"},
		NewApp:      &models.AppMetadata{AppSlug: "new"},
	}
	assert.Equal(t, "old", ci.ExistingApp.Slug)
	assert.Equal(t, "new", ci.NewApp.AppSlug)
}

func TestKetchesMetadataRoundTrip_OmitsRegistryPassword(t *testing.T) {
	appCtxs := []*models.AppContext{
		{
			App: entities.App{
				Name:             "test-app",
				Slug:             "test-app",
				AppType:          "Deployment",
				ContainerImage:   "nginx:latest",
				Replicas:         1,
				RegistryUsername: "user",
				RegistryPassword: "secret",
			},
		},
	}

	content, err := generateExport(convertAppContextsToMetadata(appCtxs), exporter.FormatKetches)
	require.NoError(t, err)
	assert.NotContains(t, content, "registry_password")

	converter := &importer.KetchesMetadataConverter{}
	apps, err := converter.Parse(content)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Empty(t, apps[0].RegistryPassword)
	assert.True(t, strings.Contains(content, "registry_username"))
}
