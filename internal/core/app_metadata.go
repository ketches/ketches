package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type AppMetadata struct {
	AppContext *models.AppContext
}

func (m *AppMetadata) BuildNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels: m.getLabels(),
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
					Labels: m.getLabels(),
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

func (m *AppMetadata) BuildRegistrySecret() *corev1.Secret {
	if m.AppContext.App.RegistryUsername == "" {
		return nil
	}

	registry := "https://index.docker.io/v1/"
	imageParts := strings.Split(m.AppContext.App.ContainerImage, "/")
	if len(imageParts) > 1 {
		if strings.Contains(imageParts[0], ".") || strings.Contains(imageParts[0], ":") || imageParts[0] == "localhost" {
			registry = imageParts[0]
		}
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", m.AppContext.App.RegistryUsername, m.AppContext.App.RegistryPassword)))
	dockerConfig := map[string]any{
		"auths": map[string]any{
			registry: map[string]any{
				"username": m.AppContext.App.RegistryUsername,
				"password": m.AppContext.App.RegistryPassword,
				"auth":     auth,
			},
		},
	}

	configJSON, _ := json.Marshal(dockerConfig)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.AppContext.App.Slug + "-registry",
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: configJSON,
		},
	}
}

func (m *AppMetadata) buildVolumes() []corev1.Volume {
	var volumes []corev1.Volume
	for _, v := range m.AppContext.Volumes {
		if v.VolumeType == app.VolumeTypePVC && m.AppContext.App.AppType != app.AppTypeStatefulSet {
			volumes = append(volumes, corev1.Volume{
				Name: v.Slug,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: v.Slug,
					},
				},
			})
		}
	}

	if len(m.AppContext.ConfigFiles) > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: "config-files",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.AppContext.App.Slug + "-config",
					},
				},
			},
		})
	}
	return volumes
}

func (m *AppMetadata) BuildConfigMap() *corev1.ConfigMap {
	data := make(map[string]string)
	for _, cf := range m.AppContext.ConfigFiles {
		data[cf.Slug] = cf.Content
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.AppContext.App.Slug + "-config",
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Data: data,
	}
}

func (m *AppMetadata) BuildPVC(v entities.AppVolume) *corev1.PersistentVolumeClaim {
	quantity := resource.MustParse(fmt.Sprintf("%dGi", v.Capacity))
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v.Slug,
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
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
					Labels: m.getLabels(),
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
			statefulSet.Spec.VolumeClaimTemplates = append(statefulSet.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      v.Slug,
					Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
					Labels:    m.getLabels(),
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", v.Capacity)),
						},
					},
				},
			})
		}
	}

	m.applySchedulingRules(&statefulSet.Spec.Template.Spec)

	return statefulSet
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
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  ev.Key,
			Value: ev.Value,
		})
	}

	for _, v := range m.AppContext.Volumes {
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
			containers = append(containers, m.buildPluginContainer(&plugin))
		}
	}
	return containers
}

func (m *AppMetadata) buildSidecarContainers() []corev1.Container {
	var containers []corev1.Container
	for _, appPlugin := range m.AppContext.AppPlugins {
		plugin, ok := m.AppContext.Plugins[appPlugin.PluginID]
		if appPlugin.Enabled && ok && plugin.PluginType == "sidecar" {
			containers = append(containers, m.buildPluginContainer(&plugin))
		}
	}
	return containers
}

func (m *AppMetadata) buildPluginContainer(plugin *entities.Plugin) corev1.Container {
	container := corev1.Container{
		Name:            plugin.Slug,
		Image:           plugin.Image,
		ImagePullPolicy: resolveImagePullPolicy(plugin.ImagePullPolicy, plugin.Image),
	}

	if plugin.Command != "" {
		container.Command = []string{"sh", "-c", plugin.Command}
	}

	container.Env = m.buildPluginEnvVars(plugin)
	container.VolumeMounts = m.buildPluginVolumeMounts()

	return container
}

func (m *AppMetadata) buildPluginEnvVars(plugin *entities.Plugin) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

	for _, ev := range m.AppContext.EnvVars {
		envVars = append(envVars, corev1.EnvVar{
			Name:  ev.Key,
			Value: ev.Value,
		})
	}

	if plugin.EnvVars != "" {
		var pluginEnvVars []models.PluginEnvVar
		if err := json.Unmarshal([]byte(plugin.EnvVars), &pluginEnvVars); err == nil {
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

func (m *AppMetadata) buildPluginVolumeMounts() []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	for _, v := range m.AppContext.Volumes {
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

func (m *AppMetadata) BuildHTTPRoute(gw entities.AppGateway) *gatewayv1.HTTPRoute {
	if !gw.Exposed {
		return nil
	}

	hostname := gatewayv1.Hostname(gw.Domain)
	pathPrefix := gatewayv1.PathMatchPathPrefix

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", m.AppContext.App.Slug, gw.Port),
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name: gatewayv1.ObjectName(EnvGatewayName(m.AppContext.EnvContext.Env.Slug)),
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{hostname},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathPrefix,
								Value: &gw.Path,
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(m.AppContext.App.Slug),
									Port: ptrPort(gatewayv1.PortNumber(gw.Port)),
								},
							},
						},
					},
				},
			},
		},
	}

	return route
}

func ptrInt32(i int32) *int32 {
	return &i
}

func ptrPort(p gatewayv1.PortNumber) *gatewayv1.PortNumber {
	return &p
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
