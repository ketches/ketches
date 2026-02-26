package exporter

import (
	"fmt"
	"strings"
	"github.com/ketches/ketches/internal/models"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

// ExportFormat defines the format of the export.
type ExportFormat string

const (
	FormatKubernetes ExportFormat = "kubernetes"
	FormatKetches    ExportFormat = "ketches"
	FormatHelm       ExportFormat = "helm"
)

// ExportGenerator defines the interface for application export generators.
type ExportGenerator interface {
	Generate(appMetadatas []models.AppMetadata) (string, error)
}

// K8sManifestGenerator generates Kubernetes YAML manifests.
type K8sManifestGenerator struct{}

// Generate generates Kubernetes YAML manifests from the given application metadata.
// Generate generates Kubernetes YAML manifests from the given application metadata.
func (g *K8sManifestGenerator) Generate(appMetadatas []models.AppMetadata) (string, error) {
	var manifests []string

	for _, app := range appMetadatas {
		// Common labels
		labels := map[string]string{
			"app": app.AppSlug,
		}

		// Container definition
		container := corev1.Container{
			Name:  "app-" + app.AppSlug,
			Image: app.ContainerImage,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{},
				Limits:   corev1.ResourceList{},
			},
		}

		// Command
		if app.ContainerCommand != "" {
			// Matches internal core/app_metadata.go implementation
			container.Command = []string{"sh", "-c", app.ContainerCommand}
		}

		// Environment variables
		if len(app.EnvVars) > 0 {
			for _, env := range app.EnvVars {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  env.Key,
					Value: env.Value,
				})
			}
		}

		// Resources
		if app.RequestCPU > 0 {
			container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", app.RequestCPU))
		}
		if app.RequestMemory > 0 {
			container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dMi", app.RequestMemory))
		}
		if app.LimitCPU > 0 {
			container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", app.LimitCPU))
		}
		if app.LimitMemory > 0 {
			container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dMi", app.LimitMemory))
		}

		// Ports
		if len(app.Gateways) > 0 {
			for _, gw := range app.Gateways {
				container.Ports = append(container.Ports, corev1.ContainerPort{
					ContainerPort: int32(gw.Port),
					Protocol:      corev1.Protocol(strings.ToUpper(gw.Protocol)),
				})
			}
		}

		// Workload (Deployment or StatefulSet)
		var workload interface{}
		objectMeta := metav1.ObjectMeta{
			Name: app.AppSlug,
		}
		podTemplate := corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{container},
			},
		}

		replicas := int32(app.Replicas)

		if app.AppType == "StatefulSet" {
			workload = &appsv1.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "StatefulSet",
				},
				ObjectMeta: objectMeta,
				Spec: appsv1.StatefulSetSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					Template: podTemplate,
					ServiceName: app.AppSlug, // Required for StatefulSet
				},
			}
		} else {
			// Default to Deployment
			workload = &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: objectMeta,
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					Template: podTemplate,
				},
			}
		}

		workloadBytes, err := yaml.Marshal(workload)
		if err != nil {
			return "", err
		}
		manifests = append(manifests, string(workloadBytes))

		// Service
		if len(app.Gateways) > 0 {
			var servicePorts []corev1.ServicePort
			for _, gw := range app.Gateways {
				servicePorts = append(servicePorts, corev1.ServicePort{
					Name:       fmt.Sprintf("port-%d", gw.Port),
					Port:       int32(gw.Port),
					TargetPort: intstr.FromInt(gw.Port),
					Protocol:   corev1.Protocol(strings.ToUpper(gw.Protocol)),
				})
			}

			service := &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: app.AppSlug,
				},
				Spec: corev1.ServiceSpec{
					Selector: labels,
					Ports:    servicePorts,
				},
			}

			serviceBytes, err := yaml.Marshal(service)
			if err != nil {
				return "", err
			}
			manifests = append(manifests, string(serviceBytes))
		}

		// ConfigMap
		if len(app.ConfigFiles) > 0 {
			data := make(map[string]string)
			for _, cf := range app.ConfigFiles {
				// Use slug or filename as key? The requirement says "data: ConfigFiles content"
				// Typically ConfigMap keys are filenames.
				// Requirement says: "metadata.name: AppSlug + "-config""
				// but doesn't specify keys. Let's use Slug as key or just "config" if not present.
				// Given `ConfigFileMetadata` has `Slug`, let's use that.
				key := cf.Slug
				if key == "" {
					key = "config"
				}
				data[key] = cf.Content
			}

			configMap := &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: app.AppSlug + "-config",
				},
				Data: data,
			}

			cmBytes, err := yaml.Marshal(configMap)
			if err != nil {
				return "", err
			}
			manifests = append(manifests, string(cmBytes))
		}
	}

	return strings.Join(manifests, "\n---\n"), nil
}

// KetchesMetadataGenerator generates Ketches metadata JSON.
type KetchesMetadataGenerator struct{}

// Generate generates Ketches metadata JSON from the given application metadata.
func (g *KetchesMetadataGenerator) Generate(appMetadatas []models.AppMetadata) (string, error) {
	return "", nil
}

// HelmChartGenerator generates a Helm Chart (ZIP file).
type HelmChartGenerator struct{}

// Generate generates a Helm Chart from the given application metadata.
func (g *HelmChartGenerator) Generate(appMetadatas []models.AppMetadata) (string, error) {
	return "", nil
}
