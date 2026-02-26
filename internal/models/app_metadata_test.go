package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppMetadata_Serialization(t *testing.T) {
	// Define a fully populated AppMetadata struct
	now := time.Now().UTC()

	// Create sub-structs first
	envVars := []EnvVarMetadata{
		{Key: "ENV_KEY", Value: "ENV_VALUE"},
	}

	volumes := []VolumeMetadata{
		{Slug: "vol-1", MountPath: "/data", Capacity: 10, VolumeType: "pvc", SubPath: "sub"},
	}

	configFiles := []ConfigFileMetadata{
		{Slug: "cfg-1", MountPath: "/cfg", Content: "config-content", FileMode: "0644"},
	}

	gateways := []GatewayMetadata{
		{Port: 8080, Protocol: "TCP", Exposed: true, Domain: "example.com", Path: "/", GatewayPort: 80, CertID: "cert-1"},
	}

	probes := []ProbeMetadata{
		{Type: "liveness", ProbeMode: "http", Enabled: true, InitialDelaySeconds: 10, HttpGetPath: "/health", HttpGetPort: 8080},
	}

	scheduling := &SchedulingMetadata{
		RuleType:     "affinity",
		NodeAffinity: "some-affinity",
	}

	autoScaling := &AutoScalingMetadata{
		MinReplicas:             1,
		MaxReplicas:             3,
		TargetCPUUtilization:    80,
		TargetMemoryUtilization: 80,
	}

	appMetadata := AppMetadata{
		AppName:          "test-app",
		AppSlug:          "test-app-slug",
		AppType:          "Deployment",
		Description:      "Test Description",
		ContainerImage:   "nginx:latest",
		ContainerCommand: "/bin/sh",
		Replicas:         2,
		RequestCPU:       100,
		RequestMemory:    128,
		LimitCPU:         200,
		LimitMemory:      256,
		RegistryUsername: "user",
		RegistryPassword: "password",
		EnvVars:          envVars,
		Volumes:          volumes,
		ConfigFiles:      configFiles,
		Gateways:         gateways,
		Probes:           probes,
		SchedulingRule:   scheduling,
		AutoScaling:      autoScaling,
		Source:           "import",
		ImportedAt:       now,
	}

	// Serialize to JSON
	data, err := json.Marshal(appMetadata)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Deserialize back
	var loadedApp AppMetadata
	err = json.Unmarshal(data, &loadedApp)
	assert.NoError(t, err)

	// Verify fields
	assert.Equal(t, appMetadata.AppName, loadedApp.AppName)
	assert.Equal(t, appMetadata.AppSlug, loadedApp.AppSlug)
	assert.Equal(t, appMetadata.Replicas, loadedApp.Replicas)
	assert.Equal(t, len(appMetadata.EnvVars), len(loadedApp.EnvVars))
	assert.Equal(t, appMetadata.EnvVars[0].Key, loadedApp.EnvVars[0].Key)
	assert.Equal(t, appMetadata.SchedulingRule.RuleType, loadedApp.SchedulingRule.RuleType)
	assert.Equal(t, appMetadata.Gateways[0].Domain, loadedApp.Gateways[0].Domain)

	// Test omitempty fields
	minimalApp := AppMetadata{
		AppName:        "min-app",
		AppSlug:        "min-app",
		AppType:        "Deployment",
		ContainerImage: "nginx",
		Replicas:       1,
		RequestCPU:     100,
		RequestMemory:  100,
		LimitCPU:       100,
		LimitMemory:    100,
		ImportedAt:     now,
	}

	minData, err := json.Marshal(minimalApp)
	assert.NoError(t, err)

	minJson := string(minData)
	assert.NotContains(t, minJson, "container_command")
	assert.NotContains(t, minJson, "registry_username")
	assert.NotContains(t, minJson, "scheduling_rule")
	assert.NotContains(t, minJson, "auto_scaling")
	assert.NotContains(t, minJson, "source")

	// imported_at should be present
	assert.Contains(t, minJson, "imported_at")
}

func TestKetchesMetadataFile_Serialization(t *testing.T) {
	now := time.Now().UTC()
	file := KetchesMetadataFile{
		Version:    "v1",
		Type:       "ketches_export",
		ExportedAt: now,
		Apps: []AppMetadata{
			{AppName: "app1"},
		},
	}

	data, err := json.Marshal(file)
	assert.NoError(t, err)

	var loadedFile KetchesMetadataFile
	err = json.Unmarshal(data, &loadedFile)
	assert.NoError(t, err)

	assert.Equal(t, file.Version, loadedFile.Version)
	assert.Equal(t, 1, len(loadedFile.Apps))
}

func TestAppMetadata_ToCreateAppRequest(t *testing.T) {
	tests := []struct {
		name     string
		metadata AppMetadata
		want     *CreateAppRequest
	}{
		{
			name: "Full conversion",
			metadata: AppMetadata{
				AppName:          "Test App",
				AppSlug:          "test-app",
				AppType:          "Deployment",
				Description:      "A test application",
				ContainerImage:   "nginx:latest",
				ContainerCommand: "/bin/sh -c 'echo hello'",
				Replicas:         3,
				RequestCPU:       100,
				RequestMemory:    256,
				LimitCPU:         200,
				LimitMemory:      512,
				RegistryUsername: "user",
				RegistryPassword: "password",
				AutoScaling: &AutoScalingMetadata{
					MinReplicas:             1,
					MaxReplicas:             5,
					TargetCPUUtilization:    80,
					TargetMemoryUtilization: 80,
				},
				SchedulingRule: &SchedulingMetadata{
					RuleType:     "NodeSelector",
					NodeSelector: "disktype=ssd",
				},
				Probes: []ProbeMetadata{
					{
						Type:                "Liveness",
						ProbeMode:           "HTTP",
						Enabled:             true,
						HttpGetPath:         "/healthz",
						HttpGetPort:         8080,
						InitialDelaySeconds: 10,
						PeriodSeconds:       10,
						TimeoutSeconds:      5,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
				},
				Gateways: []GatewayMetadata{
					{
						Port:        80,
						Protocol:    "TCP",
						Domain:      "example.com",
						Path:        "/",
						GatewayPort: 80,
						Exposed:     true,
					},
				},
			},
			want: &CreateAppRequest{
				Name:             "Test App",
				Slug:             "test-app",
				AppType:          "Deployment",
				Description:      "A test application",
				ContainerImage:   "nginx:latest",
				ContainerCommand: "/bin/sh -c 'echo hello'",
				Replicas:         3,
				RequestCPU:       100,
				RequestMemory:    256,
				LimitCPU:         200,
				LimitMemory:      512,
				RegistryUsername: "user",
				RegistryPassword: "password",
				AutoScaling: &AutoScalingSpec{
					MinReplicas:             1,
					MaxReplicas:             5,
					TargetCPUUtilization:    80,
					TargetMemoryUtilization: 80,
				},
				SchedulingRule: &SchedulingSpec{
					RuleType:     "NodeSelector",
					NodeSelector: "disktype=ssd",
				},
				Probes: []ProbeSpec{
					{
						Type:                "Liveness",
						ProbeMode:           "HTTP",
						Enabled:             true,
						HttpGetPath:         "/healthz",
						HttpGetPort:         8080,
						InitialDelaySeconds: 10,
						PeriodSeconds:       10,
						TimeoutSeconds:      5,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
				},
				Gateways: []GatewaySpec{
					{
						Port:        80,
						Protocol:    "TCP",
						Domain:      "example.com",
						Path:        "/",
						GatewayPort: 80,
						Exposed:     true,
					},
				},
			},
		},
		{
			name: "Minimal conversion",
			metadata: AppMetadata{
				AppName:        "Minimal App",
				AppSlug:        "minimal-app",
				AppType:        "StatefulSet",
				ContainerImage: "redis:alpine",
				Replicas:       1,
			},
			want: &CreateAppRequest{
				Name:           "Minimal App",
				Slug:           "minimal-app",
				AppType:        "StatefulSet",
				ContainerImage: "redis:alpine",
				Replicas:       1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.ToCreateAppRequest()

			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Slug, got.Slug)
			assert.Equal(t, tt.want.AppType, got.AppType)
			assert.Equal(t, tt.want.Description, got.Description)
			assert.Equal(t, tt.want.ContainerImage, got.ContainerImage)
			assert.Equal(t, tt.want.ContainerCommand, got.ContainerCommand)
			assert.Equal(t, tt.want.Replicas, got.Replicas)
			assert.Equal(t, tt.want.RequestCPU, got.RequestCPU)
			assert.Equal(t, tt.want.RequestMemory, got.RequestMemory)
			assert.Equal(t, tt.want.LimitCPU, got.LimitCPU)
			assert.Equal(t, tt.want.LimitMemory, got.LimitMemory)
			assert.Equal(t, tt.want.RegistryUsername, got.RegistryUsername)
			assert.Equal(t, tt.want.RegistryPassword, got.RegistryPassword)

			// Check AutoScaling
			if tt.want.AutoScaling == nil {
				assert.Nil(t, got.AutoScaling)
			} else {
				assert.NotNil(t, got.AutoScaling)
				assert.Equal(t, tt.want.AutoScaling.MinReplicas, got.AutoScaling.MinReplicas)
				assert.Equal(t, tt.want.AutoScaling.MaxReplicas, got.AutoScaling.MaxReplicas)
				assert.Equal(t, tt.want.AutoScaling.TargetCPUUtilization, got.AutoScaling.TargetCPUUtilization)
				assert.Equal(t, tt.want.AutoScaling.TargetMemoryUtilization, got.AutoScaling.TargetMemoryUtilization)
			}

			// Check SchedulingRule
			if tt.want.SchedulingRule == nil {
				assert.Nil(t, got.SchedulingRule)
			} else {
				assert.NotNil(t, got.SchedulingRule)
				assert.Equal(t, tt.want.SchedulingRule.RuleType, got.SchedulingRule.RuleType)
				assert.Equal(t, tt.want.SchedulingRule.NodeSelector, got.SchedulingRule.NodeSelector)
			}

			// Check Probes
			if len(tt.want.Probes) == 0 {
				assert.Empty(t, got.Probes)
			} else {
				assert.Len(t, got.Probes, len(tt.want.Probes))
				for i, wantProbe := range tt.want.Probes {
					assert.Equal(t, wantProbe.Type, got.Probes[i].Type)
					assert.Equal(t, wantProbe.ProbeMode, got.Probes[i].ProbeMode)
					assert.Equal(t, wantProbe.Enabled, got.Probes[i].Enabled)
					assert.Equal(t, wantProbe.HttpGetPath, got.Probes[i].HttpGetPath)
					assert.Equal(t, wantProbe.HttpGetPort, got.Probes[i].HttpGetPort)
				}
			}

			// Check Gateways
			if len(tt.want.Gateways) == 0 {
				assert.Empty(t, got.Gateways)
			} else {
				assert.Len(t, got.Gateways, len(tt.want.Gateways))
				for i, wantGateway := range tt.want.Gateways {
					assert.Equal(t, wantGateway.Port, got.Gateways[i].Port)
					assert.Equal(t, wantGateway.Protocol, got.Gateways[i].Protocol)
					assert.Equal(t, wantGateway.Domain, got.Gateways[i].Domain)
					assert.Equal(t, wantGateway.Path, got.Gateways[i].Path)
					assert.Equal(t, wantGateway.GatewayPort, got.Gateways[i].GatewayPort)
					assert.Equal(t, wantGateway.Exposed, got.Gateways[i].Exposed)
				}
			}
		})
	}
}