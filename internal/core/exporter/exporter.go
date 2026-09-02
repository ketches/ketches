package exporter

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	yamlv3 "gopkg.in/yaml.v3"

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
	FormatKubernetes    ExportFormat = "kubernetes"
	FormatKetches       ExportFormat = "ketches"
	FormatHelm          ExportFormat = "helm"
	FormatDockerCompose ExportFormat = "dockercompose"
)

// ExportGenerator defines the interface for application export generators.
type ExportGenerator interface {
	Generate(appMetadatas []models.AppMetadata) (string, error)
}

// K8sManifestGenerator generates Kubernetes YAML manifests.
type K8sManifestGenerator struct{}

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
				if env.IsSecret {
					continue
				}
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
		var workload any
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
					Template:    podTemplate,
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
				if cf.IsSecret {
					continue
				}
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

			if len(data) == 0 {
				continue
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
	metadata := models.KetchesMetadataFile{
		Version:    "v1",
		Type:       "ketches-app-export",
		Apps:       appMetadatas,
		ExportedAt: time.Now().UTC(),
	}

	bytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// HelmChartGenerator generates a Helm Chart (ZIP file).
type HelmChartGenerator struct{}

// HelmChartMetadata represents the content of Chart.yaml.
type HelmChartMetadata struct {
	APIVersion  string `json:"apiVersion"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	AppVersion  string `json:"appVersion"`
}

// HelmValues represents the content of values.yaml.
type HelmValues struct {
	ReplicaCount int `json:"replicaCount"`
	Image        struct {
		Repository string `json:"repository"`
		PullPolicy string `json:"pullPolicy"`
		Tag        string `json:"tag"`
	} `json:"image"`
	Service struct {
		Type string `json:"type"`
		Port int    `json:"port"`
	} `json:"service"`
	Resources struct {
		Limits struct {
			CPU    string `json:"cpu"`
			Memory string `json:"memory"`
		} `json:"limits"`
		Requests struct {
			CPU    string `json:"cpu"`
			Memory string `json:"memory"`
		} `json:"requests"`
	} `json:"resources"`
	Env []map[string]string `json:"env,omitempty"`
}

// Generate generates a Helm Chart from the given application metadata.
func (g *HelmChartGenerator) Generate(appMetadatas []models.AppMetadata) (string, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, app := range appMetadatas {
		// 1. Chart.yaml
		chartMeta := HelmChartMetadata{
			APIVersion:  "v2",
			Name:        app.AppSlug,
			Description: "A Helm chart for Ketches application",
			Type:        "application",
			Version:     "0.1.0",
			AppVersion:  "1.0",
		}
		chartBytes, err := yaml.Marshal(chartMeta)
		if err != nil {
			return "", err
		}
		if err := writeFileToZip(zipWriter, app.AppSlug+"/Chart.yaml", chartBytes); err != nil {
			return "", err
		}

		// 2. values.yaml
		// Extract image parts
		imageParts := strings.Split(app.ContainerImage, ":")
		repo := imageParts[0]
		tag := "latest"
		if len(imageParts) > 1 {
			tag = imageParts[1]
		}

		values := HelmValues{
			ReplicaCount: app.Replicas,
		}
		values.Image.Repository = repo
		values.Image.Tag = tag
		values.Image.PullPolicy = "IfNotPresent"
		values.Service.Type = "ClusterIP"
		values.Service.Port = 80 // Default
		if len(app.Gateways) > 0 {
			values.Service.Port = app.Gateways[0].Port
		}
		values.Resources.Limits.CPU = fmt.Sprintf("%dm", app.LimitCPU)
		values.Resources.Limits.Memory = fmt.Sprintf("%dMi", app.LimitMemory)
		values.Resources.Requests.CPU = fmt.Sprintf("%dm", app.RequestCPU)
		values.Resources.Requests.Memory = fmt.Sprintf("%dMi", app.RequestMemory)

		// Env vars
		for _, env := range app.EnvVars {
			if env.IsSecret {
				continue
			}
			values.Env = append(values.Env, map[string]string{"name": env.Key, "value": env.Value})
		}

		valuesBytes, err := yaml.Marshal(values)
		if err != nil {
			return "", err
		}
		if err := writeFileToZip(zipWriter, app.AppSlug+"/values.yaml", valuesBytes); err != nil {
			return "", err
		}

		// 3. Templates
		if err := writeFileToZip(zipWriter, app.AppSlug+"/templates/_helpers.tpl", []byte(helpersTpl)); err != nil {
			return "", err
		}

		if app.AppType == "StatefulSet" {
			if err := writeFileToZip(zipWriter, app.AppSlug+"/templates/statefulset.yaml", []byte(statefulSetTpl)); err != nil {
				return "", err
			}
		} else {
			if err := writeFileToZip(zipWriter, app.AppSlug+"/templates/deployment.yaml", []byte(deploymentTpl)); err != nil {
				return "", err
			}
		}

		if err := writeFileToZip(zipWriter, app.AppSlug+"/templates/service.yaml", []byte(serviceTpl)); err != nil {
			return "", err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func writeFileToZip(z *zip.Writer, name string, content []byte) error {
	f, err := z.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write(content)
	return err
}

const helpersTpl = `{{/*
Expand the name of the chart.
*/}}
{{- define "app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "app.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "app.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "app.labels" -}}
helm.sh/chart: {{ include "app.chart" . }}
{{ include "app.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
`

const deploymentTpl = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "app.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "app.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.service.port }}
              protocol: TCP
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          {{- if .Values.env }}
          env:
            {{- toYaml .Values.env | nindent 12 }}
          {{- end }}
`

const statefulSetTpl = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  serviceName: {{ include "app.fullname" . }}
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "app.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "app.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.service.port }}
              protocol: TCP
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          {{- if .Values.env }}
          env:
            {{- toYaml .Values.env | nindent 12 }}
          {{- end }}
`

const serviceTpl = `apiVersion: v1

kind: Service
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "app.selectorLabels" . | nindent 4 }}
`

// Docker Compose data structures
type dockerComposeFile struct {
	Version  string                          `yaml:"version"`
	Services map[string]dockerComposeService `yaml:"services"`
	Volumes  map[string]any                  `yaml:"volumes,omitempty"`
}

type dockerComposeService struct {
	Image       string                    `yaml:"image"`
	Command     []string                  `yaml:"command,omitempty"`
	Environment map[string]string         `yaml:"environment,omitempty"`
	Ports       []string                  `yaml:"ports,omitempty"`
	Volumes     []string                  `yaml:"volumes,omitempty"`
	Healthcheck *dockerComposeHealthcheck `yaml:"healthcheck,omitempty"`
	Deploy      *dockerComposeDeploy      `yaml:"deploy,omitempty"`
}

type dockerComposeHealthcheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"start_period,omitempty"`
}

type dockerComposeDeploy struct {
	Replicas int `yaml:"replicas"`
}

// DockerComposeGenerator generates Docker Compose YAML.
type DockerComposeGenerator struct{}

// Generate generates a docker-compose.yml from the given application metadata.
func (g *DockerComposeGenerator) Generate(appMetadatas []models.AppMetadata) (string, error) {
	composeFile := dockerComposeFile{
		Version:  "3.8",
		Services: make(map[string]dockerComposeService),
	}

	var topLevelVolumes map[string]any

	for _, app := range appMetadatas {
		svc := dockerComposeService{
			Image: app.ContainerImage,
		}

		// Command
		if app.ContainerCommand != "" {
			svc.Command = []string{"sh", "-c", app.ContainerCommand}
		}

		// Environment variables
		if len(app.EnvVars) > 0 {
			svc.Environment = make(map[string]string)
			for _, env := range app.EnvVars {
				if env.IsSecret {
					continue
				}
				svc.Environment[env.Key] = env.Value
			}
		}

		// Ports
		for _, gw := range app.Gateways {
			protocol := strings.ToLower(gw.Protocol)
			if protocol == "udp" {
				svc.Ports = append(svc.Ports, fmt.Sprintf("%d:%d/udp", gw.Port, gw.Port))
			} else {
				svc.Ports = append(svc.Ports, fmt.Sprintf("%d:%d", gw.Port, gw.Port))
			}
		}

		// Volumes
		for _, vol := range app.Volumes {
			svc.Volumes = append(svc.Volumes, fmt.Sprintf("%s:%s", vol.Slug, vol.MountPath))
			if topLevelVolumes == nil {
				topLevelVolumes = make(map[string]any)
			}
			topLevelVolumes[vol.Slug] = struct{}{}
		}

		// Healthcheck: first enabled liveness probe
		for _, probe := range app.Probes {
			if probe.Type == "liveness" && probe.Enabled {
				hc := &dockerComposeHealthcheck{
					Retries: probe.FailureThreshold,
				}
				if probe.PeriodSeconds > 0 {
					hc.Interval = fmt.Sprintf("%ds", probe.PeriodSeconds)
				}
				if probe.TimeoutSeconds > 0 {
					hc.Timeout = fmt.Sprintf("%ds", probe.TimeoutSeconds)
				}
				if probe.InitialDelaySeconds > 0 {
					hc.StartPeriod = fmt.Sprintf("%ds", probe.InitialDelaySeconds)
				}

				switch probe.ProbeMode {
				case "httpGet":
					hc.Test = []string{"CMD", "curl", "-f", fmt.Sprintf("http://localhost:%d%s", probe.HttpGetPort, probe.HttpGetPath)}
				case "tcpSocket":
					hc.Test = []string{"CMD", "nc", "-z", "localhost", fmt.Sprintf("%d", probe.TcpSocketPort)}
				case "exec":
					hc.Test = []string{"CMD-SHELL", probe.ExecCommand}
				}

				if len(hc.Test) == 0 {
					break // skip healthcheck for unknown probe mode
				}
				svc.Healthcheck = hc
				break
			}
		}

		// Deploy replicas
		svc.Deploy = &dockerComposeDeploy{
			Replicas: app.Replicas,
		}

		composeFile.Services[app.AppSlug] = svc
	}

	if topLevelVolumes != nil {
		composeFile.Volumes = topLevelVolumes
	}

	out, err := yamlv3.Marshal(composeFile)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
