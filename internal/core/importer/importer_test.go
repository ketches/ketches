package importer

import (
	"testing"

	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
)

// Ensure that converters implement the ImportConverter interface.
func TestImportConverterImplementation(t *testing.T) {
	var _ ImportConverter = &DockerComposeConverter{}
	var _ ImportConverter = &K8sManifestConverter{}
	var _ ImportConverter = &KetchesMetadataConverter{}
}

func TestKetchesMetadataConverter_Parse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		wantApp string
	}{
		{
			name: "Valid JSON",
			content: `{
				"version": "v1",
				"type": "ketches_export",
				"exported_at": "2023-10-27T10:00:00Z",
				"apps": [
					{
						"app_name": "test-app",
						"app_slug": "test-app",
						"app_type": "Deployment",
						"container_image": "nginx:latest",
						"replicas": 1
					}
				]
			}`,
			wantErr: false,
			wantApp: "test-app",
		},
		{
			name: "Valid YAML",
			content: `
version: v1
type: ketches_export
exported_at: 2023-10-27T10:00:00Z
apps:
  - app_name: yaml-app
    app_slug: yaml-app
    app_type: Deployment
    container_image: nginx:alpine
    replicas: 2
`,
			wantErr: false,
			wantApp: "yaml-app",
		},
		{
			name:    "Invalid Content",
			content: `invalid-content`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KetchesMetadataConverter{}
			got, err := c.Parse(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, 1)
				assert.Equal(t, tt.wantApp, got[0].AppName)
			}
		})
	}
}

func TestKetchesMetadataConverter_Validate(t *testing.T) {
	c := &KetchesMetadataConverter{}

	t.Run("Empty Apps", func(t *testing.T) {
		err := c.Validate([]models.AppMetadata{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no apps found")
	})

	t.Run("Valid Apps", func(t *testing.T) {
		err := c.Validate([]models.AppMetadata{{AppName: "test"}})
		assert.NoError(t, err)
	})
}

func TestDockerComposeConverter_Parse(t *testing.T) {
	c := &DockerComposeConverter{}

	tests := []struct {
		name    string
		content string
		want    []models.AppMetadata
		wantErr bool
	}{
		{
			name: "Basic Service",
			content: `
services:
  web:
    image: nginx:latest
`,
			want: []models.AppMetadata{
				{
					AppName:        "web",
					AppSlug:        "web",
					AppType:        "Deployment",
					ContainerImage: "nginx:latest",
					Replicas:       1,
					RequestCPU:     100,
					RequestMemory:  128,
					LimitCPU:       1000,
					LimitMemory:    512,
				},
			},
			wantErr: false,
		},
		{
			name: "Service with Resources",
			content: `
services:
  db:
    image: postgres:15
    deploy:
      replicas: 3
      resources:
        limits:
          cpu: 500m
          memory: 512Mi
        reservations:
          cpu: 100m
          memory: 256Mi
`,
			want: []models.AppMetadata{
				{
					AppName:        "db",
					AppSlug:        "db",
					AppType:        "Deployment",
					ContainerImage: "postgres:15",
					Replicas:       3,
					RequestCPU:     100,
					RequestMemory:  256,
					LimitCPU:       500,
					LimitMemory:    512,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Parse(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDockerComposeConverter_Validate(t *testing.T) {
	c := &DockerComposeConverter{}

	t.Run("Empty Metadata", func(t *testing.T) {
		err := c.Validate(nil)
		assert.Error(t, err)
	})

	t.Run("Valid Metadata", func(t *testing.T) {
		err := c.Validate([]models.AppMetadata{{AppName: "test"}})
		assert.NoError(t, err)
	})
}

func TestK8sManifestConverter_Parse(t *testing.T) {
	c := &K8sManifestConverter{}

	tests := []struct {
		name    string
		content string
		want    []models.AppMetadata
		wantErr bool
	}{
		{
			name: "Deployment",
			content: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        command: ["/bin/sh", "-c", "echo hello"]
        ports:
        - containerPort: 80
        env:
        - name: ENV_VAR
          value: "value"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
`,
			want: []models.AppMetadata{
				{
					AppName:          "nginx-deployment",
					AppSlug:          "nginx-deployment",
					AppType:          "Deployment",
					Replicas:       3,
					ContainerImage:   "nginx:1.14.2",
					ContainerCommand: "/bin/sh -c \"echo hello\"",
					Gateways:         []models.GatewayMetadata{{Port: 80, Protocol: "TCP"}},
					EnvVars:          []models.EnvVarMetadata{{Key: "ENV_VAR", Value: "value"}},
					RequestCPU:       100,
					RequestMemory:    128,
					LimitCPU:         200,
					LimitMemory:      256,
				},
			},
			wantErr: false,
		},
		{
			name: "StatefulSet",
			content: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: nginx
        image: registry.k8s.io/nginx-slim:0.8
`,
			want: []models.AppMetadata{
				{
					AppName:        "web",
					AppSlug:        "web",
					AppType:        "StatefulSet",
					Replicas:       2,
					ContainerImage: "registry.k8s.io/nginx-slim:0.8",
					// Defaults
					RequestCPU:    100,
					RequestMemory: 128,
					LimitCPU:      1000,
					LimitMemory:   512,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Parse(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if assert.Len(t, got, 1) {
					assert.Equal(t, tt.want[0].AppName, got[0].AppName)
					assert.Equal(t, tt.want[0].AppType, got[0].AppType)
					assert.Equal(t, tt.want[0].Replicas, got[0].Replicas)
					assert.Equal(t, tt.want[0].ContainerImage, got[0].ContainerImage)
					assert.Equal(t, tt.want[0].ContainerCommand, got[0].ContainerCommand)
					assert.Equal(t, tt.want[0].Gateways, got[0].Gateways)
					assert.Equal(t, tt.want[0].EnvVars, got[0].EnvVars)
					assert.Equal(t, tt.want[0].RequestCPU, got[0].RequestCPU)
					assert.Equal(t, tt.want[0].RequestMemory, got[0].RequestMemory)
					assert.Equal(t, tt.want[0].LimitCPU, got[0].LimitCPU)
					assert.Equal(t, tt.want[0].LimitMemory, got[0].LimitMemory)
				}
			}
		})
	}
}

func TestK8sManifestConverter_Validate(t *testing.T) {
	c := &K8sManifestConverter{}

	t.Run("Empty Metadata", func(t *testing.T) {
		err := c.Validate(nil)
		assert.Error(t, err)
	})

	t.Run("Valid Metadata", func(t *testing.T) {
		err := c.Validate([]models.AppMetadata{{AppName: "test"}})
		assert.NoError(t, err)
	})
}

