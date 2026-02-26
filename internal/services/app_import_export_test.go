package services

import (
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestConvertAppsToMetadata(t *testing.T) {
	now := time.Now()
	apps := []entities.App{
		{
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
			EnvVars: []entities.AppEnvVar{
				{Key: "ENV_KEY", Value: "ENV_VALUE"},
			},
			Gateways: []entities.AppGateway{
				{Port: 80, Protocol: "TCP", GatewayPort: 8080, Exposed: true, Domain: "example.com", Path: "/", CertID: "cert-id"},
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
			Base: entities.Base{CreatedAt: now},
		},
	}

	metadatas := convertAppsToMetadata(apps)

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
	assert.Equal(t, "password", meta.RegistryPassword)

	assert.Len(t, meta.EnvVars, 1)
	assert.Equal(t, "ENV_KEY", meta.EnvVars[0].Key)
	assert.Equal(t, "ENV_VALUE", meta.EnvVars[0].Value)

	assert.Len(t, meta.Gateways, 1)
	assert.Equal(t, 80, meta.Gateways[0].Port)
	assert.Equal(t, "TCP", meta.Gateways[0].Protocol)
	assert.Equal(t, 8080, meta.Gateways[0].GatewayPort)
	assert.Equal(t, true, meta.Gateways[0].Exposed)
	assert.Equal(t, "example.com", meta.Gateways[0].Domain)

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
