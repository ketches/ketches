package exporter

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
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
					AppName:          "Test App",
					AppSlug:          "test-app",
					AppType:          "Deployment",
					ContainerImage:   "nginx:latest",
					Replicas:         2,
					RequestCPU:       100,
					RequestMemory:    128,
					LimitCPU:         200,
					LimitMemory:      256,
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
func TestKetchesMetadataGenerator_Generate(t *testing.T) {
	generator := &KetchesMetadataGenerator{}

	appMetadata := []models.AppMetadata{
		{
			AppName:        "Test App",
			AppSlug:        "test-app",
			AppType:        "Deployment",
			ContainerImage: "nginx:latest",
			Replicas:       2,
		},
	}

	got, err := generator.Generate(appMetadata)
	if err != nil {
		t.Errorf("Generate() error = %v", err)
		return
	}

	if got == "" {
		t.Error("Generate() returned empty string")
		return
	}

	var metadata models.KetchesMetadataFile
	if err := json.Unmarshal([]byte(got), &metadata); err != nil {
		t.Errorf("Generate() returned invalid JSON: %v", err)
	}

	if metadata.Version != "v1" {
		t.Errorf("Expected Version v1, got %s", metadata.Version)
	}
	if metadata.Type != "ketches-app-export" {
		t.Errorf("Expected Type ketches-app-export, got %s", metadata.Type)
	}
	if len(metadata.Apps) != 1 {
		t.Errorf("Expected 1 app, got %d", len(metadata.Apps))
	}
	if len(metadata.Apps) > 0 && metadata.Apps[0].AppSlug != "test-app" {
		t.Errorf("Expected AppSlug test-app, got %s", metadata.Apps[0].AppSlug)
	}
	if metadata.ExportedAt.IsZero() {
		t.Error("Expected ExportedAt to be set")
	}
}

func TestHelmChartGenerator_Generate(t *testing.T) {
	app := models.AppMetadata{
		AppName:        "Test App",
		AppSlug:        "test-app",
		AppType:        "Deployment",
		ContainerImage: "nginx:latest",
		Replicas:       2,
		RequestCPU:     100,
		RequestMemory:  128,
		LimitCPU:       200,
		LimitMemory:    256,
		EnvVars: []models.EnvVarMetadata{
			{Key: "ENV_KEY", Value: "ENV_VALUE"},
		},
	}

	generator := &HelmChartGenerator{}
	output, err := generator.Generate([]models.AppMetadata{app})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if output == "" {
		t.Fatal("Output is empty")
	}

	// Decode base64
	zipData, err := base64.StdEncoding.DecodeString(output)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	// Verify zip content
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("Failed to read zip: %v", err)
	}

	files := make(map[string]string)
	for _, f := range zipReader.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("Failed to open file in zip: %v", err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("Failed to read file in zip: %v", err)
		}
		files[f.Name] = string(content)
	}

	expectedFiles := []string{
		"test-app/Chart.yaml",
		"test-app/values.yaml",
		"test-app/templates/deployment.yaml",
		"test-app/templates/service.yaml",
		"test-app/templates/_helpers.tpl",
	}

	for _, expected := range expectedFiles {
		if _, ok := files[expected]; !ok {
			t.Errorf("Expected file %s not found in zip", expected)
		}
	}

	// Verify Chart.yaml content (simple check)
	if _, ok := files["test-app/Chart.yaml"]; ok {
		if !strings.Contains(files["test-app/Chart.yaml"], "name: test-app") {
			t.Error("Chart.yaml does not contain correct name")
		}
	}

	// Verify values.yaml content
	if _, ok := files["test-app/values.yaml"]; ok {
		if !strings.Contains(files["test-app/values.yaml"], "replicaCount: 2") {
			t.Error("values.yaml does not contain correct replicaCount")
		}
	}

	// Test StatefulSet
	statefulApp := app
	statefulApp.AppSlug = "stateful-app"
	statefulApp.AppType = "StatefulSet"
	statefulApp.Replicas = 1

	outputStateful, err := generator.Generate([]models.AppMetadata{statefulApp})
	if err != nil {
		t.Fatalf("Generate stateful failed: %v", err)
	}

	zipDataStateful, _ := base64.StdEncoding.DecodeString(outputStateful)
	zipReaderStateful, err := zip.NewReader(bytes.NewReader(zipDataStateful), int64(len(zipDataStateful)))
	if err != nil {
		t.Fatalf("Failed to read zip stateful: %v", err)
	}

	foundStatefulSet := false
	foundDeployment := false
	for _, f := range zipReaderStateful.File {
		if f.Name == "stateful-app/templates/statefulset.yaml" {
			foundStatefulSet = true
		}
		if f.Name == "stateful-app/templates/deployment.yaml" {
			foundDeployment = true
		}
	}

	if !foundStatefulSet {
		t.Error("Expected statefulset.yaml not found in zip for StatefulSet app")
	}
	if foundDeployment {
		t.Error("Unexpected deployment.yaml found in zip for StatefulSet app")
	}

}
