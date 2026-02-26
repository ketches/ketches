package exporter

import (
	"strings"
	"testing"

	"github.com/ketches/ketches/internal/models"
)

func TestImplementations(t *testing.T) {
	// Verify that the generators implement the ExportGenerator interface
	var _ ExportGenerator = &K8sManifestGenerator{}
	var _ ExportGenerator = &KetchesMetadataGenerator{}
	var _ ExportGenerator = &HelmChartGenerator{}
}

func TestK8sManifestGenerator_Generate(t *testing.T) {
	generator := &K8sManifestGenerator{}

	tests := []struct {
		name         string
		appMetadata  []models.AppMetadata
		expectInYAML []string
	}{
		{
			name: "Deployment with Service and ConfigMap",
			appMetadata: []models.AppMetadata{
				{
					AppName:        "Test App",
					AppSlug:        "test-app",
					AppType:        "Deployment",
					ContainerImage: "nginx:latest",
					Replicas:       2,
					RequestCPU:     100,
					RequestMemory:  128,
					LimitCPU:       200,
					LimitMemory:    256,
					ContainerCommand: "nginx -g 'daemon off;'",
					EnvVars: []models.EnvVarMetadata{
						{Key: "ENV_KEY", Value: "ENV_VALUE"},
					},
					Gateways: []models.GatewayMetadata{
						{Port: 80, Protocol: "http"},
					},
					ConfigFiles: []models.ConfigFileMetadata{
						{Slug: "config-file", Content: "config-content", MountPath: "/etc/config"},
					},
				},
			},
			expectInYAML: []string{
				"kind: Deployment",
				"metadata:\n  name: test-app",
				"replicas: 2",
				"image: nginx:latest",
				"kind: Service",
				"port: 80",
				"kind: ConfigMap",
				"config-content",
				"requests:\n            cpu: 100m\n            memory: 128Mi",
				"limits:\n            cpu: 200m\n            memory: 256Mi",
			},
		},
		{
			name: "StatefulSet",
			appMetadata: []models.AppMetadata{
				{
					AppName:        "Stateful App",
					AppSlug:        "stateful-app",
					AppType:        "StatefulSet",
					ContainerImage: "postgres:latest",
					Replicas:       1,
				},
			},
			expectInYAML: []string{
				"kind: StatefulSet",
				"metadata:\n  name: stateful-app",
				"image: postgres:latest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generator.Generate(tt.appMetadata)
			if err != nil {
				t.Errorf("Generate() error = %v", err)
				return
			}

			for _, expect := range tt.expectInYAML {
				// Normalize newlines and spaces for loose comparison or use exact match if possible
				// For now, simple Contains check is good enough as a start
				if !strings.Contains(got, expect) {
					t.Errorf("Generate() missing expected content: %s", expect)
				}
			}
		})
	}
}