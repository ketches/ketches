package importer

import (
	"fmt"

	"github.com/ketches/ketches/internal/models"
	"sigs.k8s.io/yaml"
)



// ImportConverter defines the interface for converting different formats to AppMetadata.
type ImportConverter interface {
	Parse(content string) ([]models.AppMetadata, error)
	Validate(appMetadatas []models.AppMetadata) error
}

// DockerComposeConverter parses Docker Compose YAML.
type DockerComposeConverter struct{}

// Parse parses Docker Compose content.
func (c *DockerComposeConverter) Parse(content string) ([]models.AppMetadata, error) {
	// TODO: Implement Docker Compose parsing logic
	return nil, nil
}

// Validate validates the parsed metadata.
func (c *DockerComposeConverter) Validate(appMetadatas []models.AppMetadata) error {
	// TODO: Implement validation logic
	return nil
}

// K8sManifestConverter parses Kubernetes Manifest YAML.
type K8sManifestConverter struct{}

// Parse parses Kubernetes Manifest content.
func (c *K8sManifestConverter) Parse(content string) ([]models.AppMetadata, error) {
	// TODO: Implement Kubernetes Manifest parsing logic
	return nil, nil
}

// Validate validates the parsed metadata.
func (c *K8sManifestConverter) Validate(appMetadatas []models.AppMetadata) error {
	// TODO: Implement validation logic
	return nil
}

// KetchesMetadataConverter parses Ketches metadata JSON/YAML.
type KetchesMetadataConverter struct{}

// Parse parses Ketches metadata content.
func (c *KetchesMetadataConverter) Parse(content string) ([]models.AppMetadata, error) {
	var file models.KetchesMetadataFile

	// Try parsing as JSON first, then YAML (yaml.Unmarshal handles both actually, but let's be safe)
	if err := yaml.Unmarshal([]byte(content), &file); err != nil {
		return nil, fmt.Errorf("failed to parse content: %w", err)
	}

	return file.Apps, nil
}

// Validate validates the parsed metadata.
func (c *KetchesMetadataConverter) Validate(appMetadatas []models.AppMetadata) error {
	if len(appMetadatas) == 0 {
		return fmt.Errorf("no apps found in metadata")
	}
	return nil
}
