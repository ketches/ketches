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

func TestDockerComposeConverter(t *testing.T) {
	c := &DockerComposeConverter{}
	// Currently returns nil, nil as per placeholder implementation
	got, err := c.Parse("version: '3'")
	assert.NoError(t, err)
	assert.Nil(t, got)

	err = c.Validate(nil)
	assert.NoError(t, err)
}

func TestK8sManifestConverter(t *testing.T) {
	c := &K8sManifestConverter{}
	// Currently returns nil, nil as per placeholder implementation
	got, err := c.Parse("apiVersion: v1")
	assert.NoError(t, err)
	assert.Nil(t, got)

	err = c.Validate(nil)
	assert.NoError(t, err)
}
