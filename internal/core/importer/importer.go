package importer

import (
	"fmt"
	"io"
	"strings"

	appcore "github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
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
	var compose dockerCompose
	if err := yaml.Unmarshal([]byte(content), &compose); err != nil {
		return nil, appcore.WrapErrorf(err, "failed to parse docker compose: %w", err)
	}

	var apps []models.AppMetadata
	for name, service := range compose.Services {
		app := models.AppMetadata{
			AppName:        name,
			AppSlug:        name,
			AppType:        "Deployment",
			ContainerImage: service.Image,
			Replicas:       1,
			RequestCPU:     100,
			RequestMemory:  128,
			LimitCPU:       1000,
			LimitMemory:    512,
		}

		if service.Deploy != nil {
			if service.Deploy.Replicas > 0 {
				app.Replicas = service.Deploy.Replicas
			}

			if res := service.Deploy.Resources; res != nil {
				if len(res.Limits) > 0 {
					if val, ok := res.Limits["cpu"]; ok {
						if q, err := resource.ParseQuantity(val); err == nil {
							app.LimitCPU = int(q.MilliValue())
						}
					}
					if val, ok := res.Limits["memory"]; ok {
						if q, err := resource.ParseQuantity(val); err == nil {
							app.LimitMemory = int(q.Value() / (1024 * 1024))
						}
					}
				}
				if len(res.Reservations) > 0 {
					if val, ok := res.Reservations["cpu"]; ok {
						if q, err := resource.ParseQuantity(val); err == nil {
							app.RequestCPU = int(q.MilliValue())
						}
					}
					if val, ok := res.Reservations["memory"]; ok {
						if q, err := resource.ParseQuantity(val); err == nil {
							app.RequestMemory = int(q.Value() / (1024 * 1024))
						}
					}
				}
			}
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// Validate validates the parsed metadata.
func (c *DockerComposeConverter) Validate(appMetadatas []models.AppMetadata) error {
	if len(appMetadatas) == 0 {
		return appcore.NewErrorf("no apps found in metadata")
	}
	return nil
}

type dockerCompose struct {
	Services map[string]dockerService `json:"services"`
}

type dockerService struct {
	Image       string        `json:"image"`
	Command     interface{}   `json:"command,omitempty"`
	Ports       []string      `json:"ports,omitempty"`
	Environment interface{}   `json:"environment,omitempty"`
	Volumes     []string      `json:"volumes,omitempty"`
	Deploy      *dockerDeploy `json:"deploy,omitempty"`
}

type dockerDeploy struct {
	Replicas  int              `json:"replicas,omitempty"`
	Resources *dockerResources `json:"resources,omitempty"`
}

type dockerResources struct {
	Limits       map[string]string `json:"limits,omitempty"`
	Reservations map[string]string `json:"reservations,omitempty"`
}

// K8sManifestConverter parses Kubernetes Manifest YAML.
type K8sManifestConverter struct{}

// Parse parses Kubernetes Manifest content.
func (c *K8sManifestConverter) Parse(content string) ([]models.AppMetadata, error) {
	decoder := k8syaml.NewYAMLOrJSONDecoder(strings.NewReader(content), 4096)

	var apps []models.AppMetadata

	for {
		var raw map[string]interface{}
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, appcore.WrapErrorf(err, "failed to decode yaml: %w", err)
		}

		kind, ok := raw["kind"].(string)
		if !ok {
			continue
		}

		objBytes, err := yaml.Marshal(raw)
		if err != nil {
			continue
		}

		var app models.AppMetadata
		var podSpec corev1.PodSpec
		var metaName string

		switch kind {
		case "Deployment":
			var deploy appsv1.Deployment
			if err := yaml.Unmarshal(objBytes, &deploy); err != nil {
				return nil, appcore.WrapErrorf(err, "failed to unmarshal deployment: %w", err)
			}
			metaName = deploy.Name
			podSpec = deploy.Spec.Template.Spec
			app.Replicas = 1
			if deploy.Spec.Replicas != nil {
				app.Replicas = int(*deploy.Spec.Replicas)
			}
			app.AppType = "Deployment"

		case "StatefulSet":
			var sts appsv1.StatefulSet
			if err := yaml.Unmarshal(objBytes, &sts); err != nil {
				return nil, appcore.WrapErrorf(err, "failed to unmarshal statefulset: %w", err)
			}
			metaName = sts.Name
			podSpec = sts.Spec.Template.Spec
			app.Replicas = 1
			if sts.Spec.Replicas != nil {
				app.Replicas = int(*sts.Spec.Replicas)
			}
			app.AppType = "StatefulSet"

		default:
			continue
		}

		app.AppName = metaName
		app.AppSlug = metaName
		app.RequestCPU = 100
		app.RequestMemory = 128
		app.LimitCPU = 1000
		app.LimitMemory = 512

		if len(podSpec.Containers) > 0 {
			container := podSpec.Containers[0]
			app.ContainerImage = container.Image
			if len(container.Command) > 0 {
				// Helper to quote arguments if they contain spaces
				var cmds []string
				for _, cmd := range container.Command {
					if strings.Contains(cmd, " ") {
						cmds = append(cmds, fmt.Sprintf("%q", cmd))
					} else {
						cmds = append(cmds, cmd)
					}
				}
				app.ContainerCommand = strings.Join(cmds, " ")
			}

			for _, env := range container.Env {
				app.EnvVars = append(app.EnvVars, models.EnvVarMetadata{
					Key:   env.Name,
					Value: env.Value,
				})
			}

			for _, port := range container.Ports {
				protocol := string(port.Protocol)
				if protocol == "" {
					protocol = "TCP"
				}
				app.Gateways = append(app.Gateways, models.GatewayMetadata{
					Port:     int(port.ContainerPort),
					Protocol: protocol,
				})
			}

			req := container.Resources.Requests
			lim := container.Resources.Limits

			if q, ok := req[corev1.ResourceCPU]; ok {
				app.RequestCPU = int(q.MilliValue())
			}
			if q, ok := req[corev1.ResourceMemory]; ok {
				app.RequestMemory = int(q.Value() / (1024 * 1024))
			}
			if q, ok := lim[corev1.ResourceCPU]; ok {
				app.LimitCPU = int(q.MilliValue())
			}
			if q, ok := lim[corev1.ResourceMemory]; ok {
				app.LimitMemory = int(q.Value() / (1024 * 1024))
			}
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// Validate validates the parsed metadata.
func (c *K8sManifestConverter) Validate(appMetadatas []models.AppMetadata) error {
	if len(appMetadatas) == 0 {
		return appcore.NewErrorf("no apps found in metadata")
	}
	return nil
}

// KetchesMetadataConverter parses Ketches metadata JSON/YAML.
type KetchesMetadataConverter struct{}

// Parse parses Ketches metadata content.
func (c *KetchesMetadataConverter) Parse(content string) ([]models.AppMetadata, error) {
	var file models.KetchesMetadataFile

	// Try parsing as JSON first, then YAML (yaml.Unmarshal handles both actually, but let's be safe)
	if err := yaml.Unmarshal([]byte(content), &file); err != nil {
		return nil, appcore.WrapErrorf(err, "failed to parse content: %w", err)
	}

	return file.Apps, nil
}

// Validate validates the parsed metadata.
func (c *KetchesMetadataConverter) Validate(appMetadatas []models.AppMetadata) error {
	if len(appMetadatas) == 0 {
		return appcore.NewErrorf("no apps found in metadata")
	}
	return nil
}
