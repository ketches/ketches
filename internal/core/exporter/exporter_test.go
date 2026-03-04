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
	var _ ExportGenerator = &DockerComposeGenerator{}
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


func TestDockerComposeGenerator_Generate(t *testing.T) {
	generator := &DockerComposeGenerator{}

	t.Run("single app basic with image env port replicas", func(t *testing.T) {
		appMetadata := []models.AppMetadata{
			{
				AppName:        "Web App",
				AppSlug:        "web-app",
				AppType:        "Deployment",
				ContainerImage: "nginx:1.25",
				Replicas:       3,
				EnvVars: []models.EnvVarMetadata{
					{Key: "NODE_ENV", Value: "production"},
					{Key: "PORT", Value: "8080"},
				},
				Gateways: []models.GatewayMetadata{
					{Port: 80, Protocol: "http"},
				},
			},
		}

		got, err := generator.Generate(appMetadata)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		// Verify YAML structure
		expected := []string{
			"version: \"3.8\"",
			"web-app:",
			"image: nginx:1.25",
			"NODE_ENV: production",
			"PORT: \"8080\"",
			"80:80",
			"replicas: 3",
		}
		for _, exp := range expected {
			if !strings.Contains(got, exp) {
				t.Errorf("Generate() missing expected content %q in output:\n%s", exp, got)
			}
		}
	})

	t.Run("app with volumes", func(t *testing.T) {
		appMetadata := []models.AppMetadata{
			{
				AppName:        "DB App",
				AppSlug:        "db-app",
				AppType:        "StatefulSet",
				ContainerImage: "postgres:15",
				Replicas:       1,
				Volumes: []models.VolumeMetadata{
					{Slug: "data-vol", MountPath: "/var/lib/postgresql/data"},
					{Slug: "backup-vol", MountPath: "/backups"},
				},
			},
		}

		got, err := generator.Generate(appMetadata)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		// Verify service volumes
		expected := []string{
			"data-vol:/var/lib/postgresql/data",
			"backup-vol:/backups",
		}
		for _, exp := range expected {
			if !strings.Contains(got, exp) {
				t.Errorf("Generate() missing volume %q in output:\n%s", exp, got)
			}
		}

		// Verify top-level volumes section
		if !strings.Contains(got, "volumes:") {
			t.Errorf("Generate() missing top-level volumes section in output:\n%s", got)
		}
		if !strings.Contains(got, "data-vol: {}") {
			t.Errorf("Generate() missing top-level volume 'data-vol: {}' in output:\n%s", got)
		}
	})

	t.Run("app with liveness probe httpGet", func(t *testing.T) {
		appMetadata := []models.AppMetadata{
			{
				AppName:        "Health App",
				AppSlug:        "health-app",
				AppType:        "Deployment",
				ContainerImage: "myapp:latest",
				Replicas:       2,
				Gateways: []models.GatewayMetadata{
					{Port: 8080, Protocol: "http"},
				},
				Probes: []models.ProbeMetadata{
					{
						Type:                "liveness",
						ProbeMode:           "httpGet",
						Enabled:             true,
						HttpGetPath:         "/healthz",
						HttpGetPort:         8080,
						PeriodSeconds:       10,
						TimeoutSeconds:      5,
						FailureThreshold:    3,
						InitialDelaySeconds: 30,
					},
				},
			},
		}

		got, err := generator.Generate(appMetadata)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		expected := []string{
			"healthcheck:",
			"curl",
			"-f",
			"http://localhost:8080/healthz",
			"interval: 10s",
			"timeout: 5s",
			"retries: 3",
			"start_period: 30s",
		}
		for _, exp := range expected {
			if !strings.Contains(got, exp) {
				t.Errorf("Generate() missing expected content %q in output:\n%s", exp, got)
			}
		}
	})


	t.Run("app with liveness probe tcpSocket", func(t *testing.T) {
		appMetadata := []models.AppMetadata{
			{
				AppSlug:        "tcp-app",
				ContainerImage: "redis:7",
				Replicas:       1,
				Probes: []models.ProbeMetadata{
					{
						Type:             "liveness",
						ProbeMode:        "tcpSocket",
						Enabled:          true,
						TcpSocketPort:    6379,
						PeriodSeconds:    10,
						TimeoutSeconds:   3,
						FailureThreshold: 3,
					},
				},
			},
		}

		got, err := generator.Generate(appMetadata)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		expected := []string{
			"healthcheck:",
			"CMD",
			"nc",
			"6379",
			"interval: 10s",
			"timeout: 3s",
		}
		for _, exp := range expected {
			if !strings.Contains(got, exp) {
				t.Errorf("Generate() missing expected content %q in output:\n%s", exp, got)
			}
		}
	})

	t.Run("app with liveness probe exec", func(t *testing.T) {
		appMetadata := []models.AppMetadata{
			{
				AppSlug:        "exec-app",
				ContainerImage: "mysql:8",
				Replicas:       1,
				Probes: []models.ProbeMetadata{
					{
						Type:             "liveness",
						ProbeMode:        "exec",
						Enabled:          true,
						ExecCommand:      "mysqladmin ping -h localhost",
						PeriodSeconds:    20,
						TimeoutSeconds:   5,
						FailureThreshold: 3,
					},
				},
			},
		}

		got, err := generator.Generate(appMetadata)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		expected := []string{
			"healthcheck:",
			"CMD-SHELL",
			"mysqladmin",
			"interval: 20s",
		}
		for _, exp := range expected {
			if !strings.Contains(got, exp) {
				t.Errorf("Generate() missing expected content %q in output:\n%s", exp, got)
			}
		}
	})

	t.Run("zero replicas", func(t *testing.T) {
		appMetadata := []models.AppMetadata{
			{
				AppSlug:        "scaled-down",
				ContainerImage: "myapp:1.0",
				Replicas:       0,
			},
		}

		got, err := generator.Generate(appMetadata)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		expected := []string{
			"deploy:",
			"replicas: 0",
		}
		for _, exp := range expected {
			if !strings.Contains(got, exp) {
				t.Errorf("Generate() missing expected content %q in output:\n%s", exp, got)
			}
		}
	})
	t.Run("multi-app single file", func(t *testing.T) {
		appMetadata := []models.AppMetadata{
			{
				AppName:        "Frontend",
				AppSlug:        "frontend",
				AppType:        "Deployment",
				ContainerImage: "react-app:latest",
				Replicas:       2,
				Gateways: []models.GatewayMetadata{
					{Port: 3000, Protocol: "http"},
				},
			},
			{
				AppName:          "Backend",
				AppSlug:          "backend",
				AppType:          "Deployment",
				ContainerImage:   "go-api:latest",
				Replicas:         1,
				ContainerCommand: "./server --port=8080",
				Gateways: []models.GatewayMetadata{
					{Port: 8080, Protocol: "http"},
				},
			},
		}

		got, err := generator.Generate(appMetadata)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		// Both services should be present
		if !strings.Contains(got, "frontend:") {
			t.Errorf("Generate() missing 'frontend' service in output:\n%s", got)
		}
		if !strings.Contains(got, "backend:") {
			t.Errorf("Generate() missing 'backend' service in output:\n%s", got)
		}
		// Backend should have command
		if !strings.Contains(got, "./server --port=8080") {
			t.Errorf("Generate() missing command in output:\n%s", got)
		}
		// Should be a single compose file (one version line)
		if strings.Count(got, "version:") != 1 {
			t.Errorf("Generate() expected exactly one 'version:' in output, got %d", strings.Count(got, "version:"))
		}
	})
}