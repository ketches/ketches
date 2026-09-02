package core

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const configRevisionAnnotation = "ketches.cn/config-revision"

type AppMetadata struct {
	AppContext     *models.AppContext
	configRevision string
}

type configFileRevision struct {
	Slug      string `json:"slug"`
	MountPath string `json:"mount_path"`
	IsSecret  bool   `json:"is_secret"`
}

func (m *AppMetadata) BuildNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels: map[string]string{
				kube.LabelEnvID:       m.AppContext.EnvContext.Env.ID,
				kube.LabelEnvSlug:     m.AppContext.EnvContext.Env.Slug,
				kube.LabelProjectID:   m.AppContext.EnvContext.Project.ID,
				kube.LabelProjectSlug: m.AppContext.EnvContext.Project.Slug,
				kube.LabelManagedBy:   "true",
			},
		},
	}
}

func (m *AppMetadata) BuildDeployment() *appsv1.Deployment {
	replicas := int32(m.AppContext.App.Replicas)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.AppContext.App.Slug,
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: m.getSelectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      m.getLabels(),
					Annotations: m.configRevisionAnnotations(),
				},
				Spec: corev1.PodSpec{
					InitContainers: m.buildInitContainers(),
					Containers: append(
						[]corev1.Container{m.buildContainer()},
						m.buildSidecarContainers()...,
					),
					Volumes: m.buildVolumes(),
				},
			},
		},
	}

	if m.AppContext.App.RegistryUsername != "" {
		deployment.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: m.AppContext.App.Slug + "-registry"},
		}
	}

	m.applySchedulingRules(&deployment.Spec.Template.Spec)

	return deployment
}

func (m *AppMetadata) BuildConfigRevision() (string, error) {
	var envSecretData map[string][]byte
	if m.hasSecretEnvVars() {
		envSecret, err := m.BuildEnvSecret()
		if err != nil {
			return "", err
		}
		envSecretData = envSecret.Data
	}
	return m.buildConfigRevisionWithEnvSecretData(envSecretData)
}

func (m *AppMetadata) buildConfigRevisionWithEnvSecretData(envSecretData map[string][]byte) (string, error) {
	var configMapData map[string]string
	if m.hasNonSecretConfigFiles() {
		configMapData = m.BuildConfigMap().Data
	}

	var configSecretData map[string][]byte
	if m.hasSecretConfigFiles() {
		configSecret, err := m.BuildConfigSecret()
		if err != nil {
			return "", err
		}
		configSecretData = configSecret.Data
	}

	configFiles := make([]configFileRevision, 0, len(m.AppContext.ConfigFiles))
	for _, configFile := range m.AppContext.ConfigFiles {
		configFiles = append(configFiles, configFileRevision{
			Slug:      configFile.Slug,
			MountPath: configFile.MountPath,
			IsSecret:  configFile.IsSecret,
		})
	}
	sort.Slice(configFiles, func(i, j int) bool {
		if configFiles[i].Slug != configFiles[j].Slug {
			return configFiles[i].Slug < configFiles[j].Slug
		}
		if configFiles[i].MountPath != configFiles[j].MountPath {
			return configFiles[i].MountPath < configFiles[j].MountPath
		}
		return !configFiles[i].IsSecret && configFiles[j].IsSecret
	})

	payload := struct {
		ConfigFiles      []configFileRevision
		ConfigMapData    map[string]string
		ConfigSecretData map[string][]byte
		EnvSecretData    map[string][]byte
	}{
		ConfigFiles:      configFiles,
		ConfigMapData:    configMapData,
		ConfigSecretData: configSecretData,
		EnvSecretData:    envSecretData,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", app.WrapErrorf(err, "encode app configuration revision")
	}
	revision := sha256.Sum256(encoded)
	return hex.EncodeToString(revision[:]), nil
}

func (m *AppMetadata) configRevisionAnnotations() map[string]string {
	if m.configRevision == "" {
		return nil
	}
	return map[string]string{configRevisionAnnotation: m.configRevision}
}

func (m *AppMetadata) BuildRegistrySecret() (*corev1.Secret, error) {
	if m.AppContext.App.RegistryUsername == "" {
		return nil, nil
	}

	plaintextRegistryPassword, err := secrets.DecryptString(m.AppContext.App.RegistryPassword)
	if err != nil {
		return nil, app.WrapErrorf(err, "decrypt registry password: %w", err)
	}

	registry := "https://index.docker.io/v1/"
	imageParts := strings.Split(m.AppContext.App.ContainerImage, "/")
	if len(imageParts) > 1 {
		if strings.Contains(imageParts[0], ".") || strings.Contains(imageParts[0], ":") || imageParts[0] == "localhost" {
			registry = imageParts[0]
		}
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", m.AppContext.App.RegistryUsername, plaintextRegistryPassword)))
	dockerConfig := map[string]any{
		"auths": map[string]any{
			registry: map[string]any{
				"username": m.AppContext.App.RegistryUsername,
				"password": plaintextRegistryPassword,
				"auth":     auth,
			},
		},
	}

	configJSON, _ := json.Marshal(dockerConfig)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.AppContext.App.Slug + "-registry",
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    withAppComponent(m.getLabels(), "registry-secret"),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: configJSON,
		},
	}, nil
}

func (m *AppMetadata) buildVolumes() []corev1.Volume {
	var volumes []corev1.Volume
	for _, v := range m.AppContext.Volumes {
		volume := corev1.Volume{Name: v.Slug}
		switch v.VolumeType {
		case app.VolumeTypePVC:
			if m.AppContext.App.AppType == app.AppTypeStatefulSet {
				continue
			}
			volume.VolumeSource.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{ClaimName: v.Slug}
		case app.VolumeTypeEmptyDir:
			volume.VolumeSource.EmptyDir = &corev1.EmptyDirVolumeSource{}
		case app.VolumeTypeHostPath:
			hostPathType := corev1.HostPathDirectoryOrCreate
			volume.VolumeSource.HostPath = &corev1.HostPathVolumeSource{Path: v.HostPath, Type: &hostPathType}
		default:
			continue
		}
		volumes = append(volumes, volume)
	}

	if len(m.AppContext.ConfigFiles) > 0 {
		projectedSources := []corev1.VolumeProjection{}
		if m.hasNonSecretConfigFiles() {
			projectedSources = append(projectedSources, corev1.VolumeProjection{ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{Name: m.AppContext.App.Slug + "-config"},
			}})
		}
		if m.hasSecretConfigFiles() {
			projectedSources = append(projectedSources, corev1.VolumeProjection{Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{Name: m.AppContext.App.Slug + "-config-secret"},
			}})
		}
		volumes = append(volumes, corev1.Volume{
			Name: "config-files",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{Sources: projectedSources},
			},
		})
	}
	return volumes
}

func (m *AppMetadata) BuildConfigMap() *corev1.ConfigMap {
	data := make(map[string]string)
	for _, cf := range m.AppContext.ConfigFiles {
		if !cf.IsSecret {
			data[cf.Slug] = cf.Content
		}
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.AppContext.App.Slug + "-config",
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    withAppComponent(m.getLabels(), "config"),
		},
		Data: data,
	}
}

func (m *AppMetadata) BuildConfigSecret() (*corev1.Secret, error) {
	data := make(map[string][]byte)
	for _, cf := range m.AppContext.ConfigFiles {
		if !cf.IsSecret {
			continue
		}
		content, err := secrets.DecryptStringCompatible(cf.Content)
		if err != nil {
			return nil, app.WrapErrorf(err, "decrypt secret config file %s: %w", cf.Slug, err)
		}
		data[cf.Slug] = []byte(content)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name: m.AppContext.App.Slug + "-config-secret", Namespace: m.AppContext.EnvContext.Env.ClusterNamespace, Labels: withAppComponent(m.getLabels(), "config-secret"),
	}, Type: corev1.SecretTypeOpaque, Data: data}, nil
}

func (m *AppMetadata) BuildEnvSecret() (*corev1.Secret, error) {
	data := make(map[string][]byte)
	for _, ev := range m.AppContext.EnvVars {
		if !ev.IsSecret {
			continue
		}
		value, err := secrets.DecryptStringCompatible(ev.Value)
		if err != nil {
			return nil, app.WrapErrorf(err, "decrypt secret environment variable %s: %w", ev.Key, err)
		}
		data[ev.Key] = []byte(value)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name: m.AppContext.App.Slug + "-env-secret", Namespace: m.AppContext.EnvContext.Env.ClusterNamespace, Labels: withAppComponent(m.getLabels(), "env-secret"),
	}, Type: corev1.SecretTypeOpaque, Data: data}, nil
}

func (m *AppMetadata) hasNonSecretConfigFiles() bool {
	for _, cf := range m.AppContext.ConfigFiles {
		if !cf.IsSecret {
			return true
		}
	}
	return false
}

func (m *AppMetadata) hasSecretConfigFiles() bool {
	for _, cf := range m.AppContext.ConfigFiles {
		if cf.IsSecret {
			return true
		}
	}
	return false
}

func (m *AppMetadata) hasSecretEnvVars() bool {
	for _, ev := range m.AppContext.EnvVars {
		if ev.IsSecret {
			return true
		}
	}
	return false
}

func (m *AppMetadata) BuildPVC(v entities.AppVolume) *corev1.PersistentVolumeClaim {
	quantity := resource.MustParse(fmt.Sprintf("%dGi", v.Capacity))
	volumeMode := corev1.PersistentVolumeFilesystem
	if v.VolumeMode == string(corev1.PersistentVolumeBlock) {
		volumeMode = corev1.PersistentVolumeBlock
	}
	var storageClassName *string
	if strings.TrimSpace(v.StorageClass) != "" {
		storageClass := strings.TrimSpace(v.StorageClass)
		storageClassName = &storageClass
	}
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v.Slug,
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      parsePersistentVolumeAccessModes(v.AccessModes),
			StorageClassName: storageClassName,
			VolumeMode:       &volumeMode,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
		},
	}
}

func (m *AppMetadata) BuildStatefulSet() *appsv1.StatefulSet {
	replicas := int32(m.AppContext.App.Replicas)
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.AppContext.App.Slug,
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: m.getSelectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      m.getLabels(),
					Annotations: m.configRevisionAnnotations(),
				},
				Spec: corev1.PodSpec{
					InitContainers: m.buildInitContainers(),
					Containers: append(
						[]corev1.Container{m.buildContainer()},
						m.buildSidecarContainers()...,
					),
					Volumes: m.buildVolumes(),
				},
			},
		},
	}

	if m.AppContext.App.RegistryUsername != "" {
		statefulSet.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: m.AppContext.App.Slug + "-registry"},
		}
	}

	for _, v := range m.AppContext.Volumes {
		if v.VolumeType == app.VolumeTypePVC {
			claim := m.BuildPVC(v)
			statefulSet.Spec.VolumeClaimTemplates = append(statefulSet.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{
				ObjectMeta: claim.ObjectMeta,
				Spec:       claim.Spec,
			})
		}
	}

	m.applySchedulingRules(&statefulSet.Spec.Template.Spec)

	return statefulSet
}

func parsePersistentVolumeAccessModes(value string) []corev1.PersistentVolumeAccessMode {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	modes := make([]corev1.PersistentVolumeAccessMode, 0, len(parts))
	seen := make(map[corev1.PersistentVolumeAccessMode]struct{}, len(parts))
	for _, part := range parts {
		mode := corev1.PersistentVolumeAccessMode(strings.TrimSpace(part))
		switch mode {
		case corev1.ReadWriteOnce, corev1.ReadOnlyMany, corev1.ReadWriteMany, corev1.ReadWriteOncePod:
			if _, ok := seen[mode]; ok {
				continue
			}
			seen[mode] = struct{}{}
			modes = append(modes, mode)
		}
	}
	if len(modes) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	return modes
}

func (m *AppMetadata) getLabels() map[string]string {
	return map[string]string{
		kube.LabelAppID:       m.AppContext.App.ID,
		kube.LabelAppSlug:     m.AppContext.App.Slug,
		kube.LabelEnvID:       m.AppContext.EnvContext.Env.ID,
		kube.LabelEnvSlug:     m.AppContext.EnvContext.Env.Slug,
		kube.LabelProjectID:   m.AppContext.EnvContext.Project.ID,
		kube.LabelProjectSlug: m.AppContext.EnvContext.Project.Slug,
		kube.LabelManagedBy:   "true",
	}
}

func withAppComponent(labels map[string]string, component string) map[string]string {
	result := make(map[string]string, len(labels)+1)
	for key, value := range labels {
		result[key] = value
	}
	result[kube.LabelComponent] = component
	return result
}

func (m *AppMetadata) getSelectorLabels() map[string]string {
	return map[string]string{
		kube.LabelAppID:   m.AppContext.App.ID,
		kube.LabelAppSlug: m.AppContext.App.Slug,
	}
}

func (m *AppMetadata) buildContainer() corev1.Container {
	container := corev1.Container{
		Name:            "app-" + m.AppContext.App.Slug,
		Image:           m.AppContext.App.ContainerImage,
		ImagePullPolicy: resolveImagePullPolicy(m.AppContext.App.ImagePullPolicy, m.AppContext.App.ContainerImage),
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", m.AppContext.App.RequestCPU)),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", m.AppContext.App.RequestMemory)),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", m.AppContext.App.LimitCPU)),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", m.AppContext.App.LimitMemory)),
			},
		},
	}

	if m.AppContext.App.ContainerCommand != "" {
		container.Command = []string{"sh", "-c", m.AppContext.App.ContainerCommand}
	}

	for _, ev := range m.AppContext.EnvVars {
		if ev.IsSecret {
			container.Env = append(container.Env, corev1.EnvVar{Name: ev.Key, ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: m.AppContext.App.Slug + "-env-secret"}, Key: ev.Key},
			}})
		} else {
			container.Env = append(container.Env, corev1.EnvVar{Name: ev.Key, Value: ev.Value})
		}
	}

	for _, v := range m.AppContext.Volumes {
		if isBlockVolume(v) {
			container.VolumeDevices = append(container.VolumeDevices, corev1.VolumeDevice{
				Name:       v.Slug,
				DevicePath: v.MountPath,
			})
			continue
		}
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      v.Slug,
			MountPath: v.MountPath,
			SubPath:   v.SubPath,
		})
	}

	for _, cf := range m.AppContext.ConfigFiles {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "config-files",
			MountPath: cf.MountPath,
			SubPath:   cf.Slug,
		})
	}

	for _, p := range m.AppContext.Probes {
		probe := &corev1.Probe{
			InitialDelaySeconds: int32(p.InitialDelaySeconds),
			PeriodSeconds:       int32(p.PeriodSeconds),
			TimeoutSeconds:      int32(p.TimeoutSeconds),
			SuccessThreshold:    int32(p.SuccessThreshold),
			FailureThreshold:    int32(p.FailureThreshold),
		}

		switch p.ProbeMode {
		case "httpGet":
			probe.HTTPGet = &corev1.HTTPGetAction{
				Path: p.HttpGetPath,
				Port: intstr.FromInt(p.HttpGetPort),
			}
		case "tcpSocket":
			probe.TCPSocket = &corev1.TCPSocketAction{
				Port: intstr.FromInt(p.TcpSocketPort),
			}
		case "exec":
			probe.Exec = &corev1.ExecAction{
				Command: []string{"sh", "-c", p.ExecCommand},
			}
		}

		switch p.Type {
		case "liveness":
			container.LivenessProbe = probe
		case "readiness":
			container.ReadinessProbe = probe
		case "startup":
			container.StartupProbe = probe
		}
	}

	return container
}

func (m *AppMetadata) buildInitContainers() []corev1.Container {
	var containers []corev1.Container
	for _, appPlugin := range m.AppContext.AppPlugins {
		plugin, ok := m.AppContext.Plugins[appPlugin.PluginID]
		if appPlugin.Enabled && ok && plugin.PluginType == "init" {
			containers = append(containers, m.buildPluginContainer(&plugin, &appPlugin))
		}
	}
	return containers
}

func (m *AppMetadata) buildSidecarContainers() []corev1.Container {
	var containers []corev1.Container
	for _, appPlugin := range m.AppContext.AppPlugins {
		plugin, ok := m.AppContext.Plugins[appPlugin.PluginID]
		if appPlugin.Enabled && ok && plugin.PluginType == "sidecar" {
			containers = append(containers, m.buildPluginContainer(&plugin, &appPlugin))
		}
	}
	return containers
}

func (m *AppMetadata) buildPluginContainer(plugin *entities.Plugin, appPlugin *entities.AppPlugin) corev1.Container {
	container := corev1.Container{
		Name:            plugin.Slug,
		Image:           plugin.Image,
		ImagePullPolicy: resolveImagePullPolicy(plugin.ImagePullPolicy, plugin.Image),
	}

	if plugin.Command != "" {
		container.Command = []string{"sh", "-c", plugin.Command}
	}

	container.Env = m.buildPluginEnvVars(plugin, appPlugin)
	container.VolumeMounts = m.buildPluginVolumeMounts()
	container.VolumeDevices = m.buildPluginVolumeDevices()
	m.applyPluginResources(&container, appPlugin)

	return container
}

func (m *AppMetadata) buildPluginEnvVars(plugin *entities.Plugin, appPlugin *entities.AppPlugin) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

	for _, ev := range m.AppContext.EnvVars {
		if ev.IsSecret {
			envVars = append(envVars, corev1.EnvVar{Name: ev.Key, ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: m.AppContext.App.Slug + "-env-secret"}, Key: ev.Key},
			}})
		} else {
			envVars = append(envVars, corev1.EnvVar{Name: ev.Key, Value: ev.Value})
		}
	}

	rawPluginEnvVars := plugin.EnvVars
	if appPlugin != nil {
		rawAppPluginEnvVars := strings.TrimSpace(appPlugin.EnvVars)
		if rawAppPluginEnvVars != "" && rawAppPluginEnvVars != "null" {
			rawPluginEnvVars = appPlugin.EnvVars
		}
	}

	if rawPluginEnvVars != "" {
		var pluginEnvVars []models.PluginEnvVar
		if err := json.Unmarshal([]byte(rawPluginEnvVars), &pluginEnvVars); err == nil {
			for _, pev := range pluginEnvVars {
				envVars = append(envVars, corev1.EnvVar{
					Name:  pev.Key,
					Value: pev.Value,
				})
			}
		}
	}

	return envVars
}

func (m *AppMetadata) applyPluginResources(container *corev1.Container, appPlugin *entities.AppPlugin) {
	if appPlugin == nil {
		return
	}

	normalizedAppPlugin := entities.NormalizeAppPluginResources(*appPlugin)
	resources := corev1.ResourceRequirements{}
	if normalizedAppPlugin.RequestCPU > 0 || normalizedAppPlugin.RequestMemory > 0 {
		resources.Requests = corev1.ResourceList{}
		if normalizedAppPlugin.RequestCPU > 0 {
			resources.Requests[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", normalizedAppPlugin.RequestCPU))
		}
		if normalizedAppPlugin.RequestMemory > 0 {
			resources.Requests[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dMi", normalizedAppPlugin.RequestMemory))
		}
	}
	if normalizedAppPlugin.LimitCPU > 0 || normalizedAppPlugin.LimitMemory > 0 {
		resources.Limits = corev1.ResourceList{}
		if normalizedAppPlugin.LimitCPU > 0 {
			resources.Limits[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", normalizedAppPlugin.LimitCPU))
		}
		if normalizedAppPlugin.LimitMemory > 0 {
			resources.Limits[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dMi", normalizedAppPlugin.LimitMemory))
		}
	}

	if len(resources.Requests) == 0 && len(resources.Limits) == 0 {
		return
	}

	container.Resources = resources
}

func (m *AppMetadata) buildPluginVolumeMounts() []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	for _, v := range m.AppContext.Volumes {
		if isBlockVolume(v) {
			continue
		}
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      v.Slug,
			MountPath: v.MountPath,
			SubPath:   v.SubPath,
		})
	}

	for _, cf := range m.AppContext.ConfigFiles {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "config-files",
			MountPath: cf.MountPath,
			SubPath:   cf.Slug,
		})
	}

	return volumeMounts
}

func (m *AppMetadata) buildPluginVolumeDevices() []corev1.VolumeDevice {
	var volumeDevices []corev1.VolumeDevice
	for _, volume := range m.AppContext.Volumes {
		if isBlockVolume(volume) {
			volumeDevices = append(volumeDevices, corev1.VolumeDevice{Name: volume.Slug, DevicePath: volume.MountPath})
		}
	}
	return volumeDevices
}

func isBlockVolume(volume entities.AppVolume) bool {
	return volume.VolumeType == app.VolumeTypePVC && volume.VolumeMode == string(corev1.PersistentVolumeBlock)
}

func (m *AppMetadata) applySchedulingRules(podSpec *corev1.PodSpec) {
	if m.AppContext.SchedulingRule == nil {
		return
	}

	rule := m.AppContext.SchedulingRule
	if rule.NodeName != "" {
		podSpec.NodeName = rule.NodeName
	}

	if rule.NodeSelector != "" {
		var selector map[string]string
		if err := json.Unmarshal([]byte(rule.NodeSelector), &selector); err == nil {
			podSpec.NodeSelector = selector
		}
	}

	if rule.NodeAffinity != "" {
		var affinity corev1.NodeAffinity
		if err := json.Unmarshal([]byte(rule.NodeAffinity), &affinity); err == nil {
			podSpec.Affinity = &corev1.Affinity{
				NodeAffinity: &affinity,
			}
		}
	}

	if rule.Tolerations != "" {
		var tolerations []corev1.Toleration
		if err := json.Unmarshal([]byte(rule.Tolerations), &tolerations); err == nil {
			podSpec.Tolerations = tolerations
		}
	}
}

func (m *AppMetadata) BuildHorizontalPodAutoscaler() *autoscalingv2.HorizontalPodAutoscaler {
	if m.AppContext.AutoScaling == nil {
		return nil
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.AppContext.App.Slug,
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       m.AppContext.App.AppType,
				Name:       m.AppContext.App.Slug,
			},
			MinReplicas: ptrInt32(int32(m.AppContext.AutoScaling.MinReplicas)),
			MaxReplicas: int32(m.AppContext.AutoScaling.MaxReplicas),
		},
	}

	if m.AppContext.AutoScaling.TargetCPUUtilization > 0 {
		hpa.Spec.Metrics = append(hpa.Spec.Metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: ptrInt32(int32(m.AppContext.AutoScaling.TargetCPUUtilization)),
				},
			},
		})
	}

	if m.AppContext.AutoScaling.TargetMemoryUtilization > 0 {
		hpa.Spec.Metrics = append(hpa.Spec.Metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: ptrInt32(int32(m.AppContext.AutoScaling.TargetMemoryUtilization)),
				},
			},
		})
	}

	return hpa
}

func ptrInt32(i int32) *int32 {
	return &i
}

func ptrPort(p gatewayv1.PortNumber) *gatewayv1.PortNumber {
	return &p
}

func ptrGatewayNamespace(namespace gatewayv1.Namespace) *gatewayv1.Namespace {
	return &namespace
}

func ptrSectionName(name gatewayv1.SectionName) *gatewayv1.SectionName {
	return &name
}

func resolveImagePullPolicy(policy, image string) corev1.PullPolicy {
	if policy != "" {
		return corev1.PullPolicy(policy)
	}
	tag := extractImageTag(image)
	if tag == "latest" || tag == "" {
		return corev1.PullAlways
	}
	return corev1.PullIfNotPresent
}

func extractImageTag(image string) string {
	withoutDigest := strings.SplitN(image, "@", 2)[0]
	parts := strings.SplitN(withoutDigest, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
