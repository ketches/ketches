package core

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppMetadata struct {
	AppID            string                   `json:"appId"`
	AppSlug          string                   `json:"appSlug"`
	DisplayName      string                   `json:"displayName"`
	Description      string                   `json:"description"`
	WorkloadType     string                   `json:"workloadType"`
	RequestCPU       int32                    `json:"requestCPU"`
	RequestMemory    int32                    `json:"requestMemory"`
	LimitCPU         int32                    `json:"limitCPU"`
	LimitMemory      int32                    `json:"limitMemory"`
	Replicas         int32                    `json:"replicas"`
	ContainerImage   string                   `json:"containerImage"`
	RegistryUsername string                   `json:"registryUsername"`
	RegistryPassword string                   `json:"registryPassword"`
	ContainerCommand string                   `json:"containerCommand"`
	EnvVars          []AppMetadataEnvVar      `json:"envVars,omitempty"`
	Volumes          []AppMetadataVolume      `json:"volumes,omitempty"`
	Ports            []AppMetadataPort        `json:"ports,omitempty"`
	HealthChecks     []AppMetadataHealthCheck `json:"healthChecks,omitempty"`
	Edition          string                   `json:"edition,omitempty"`
	EnvID            string                   `json:"envId,omitempty"`
	EnvSlug          string                   `json:"envSlug,omitempty"`
	ProjectID        string                   `json:"projectId,omitempty"`
	ProjectSlug      string                   `json:"projectSlug,omitempty"`
	ClusterNamespace string                   `json:"clusterNamespace"`
}

type AppMetadataEnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AppMetadataPort struct {
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

type AppMetadataVolume struct {
	Slug         string   `json:"slug"`
	MountPath    string   `json:"mountPath"`
	SubPath      string   `json:"subPath,omitempty"`
	StorageClass string   `json:"storageClass,omitempty"`
	AccessModes  []string `json:"accessModes,omitempty"`
	VolumeType   string   `json:"volumeType"`
	Capacity     int      `json:"capacity"`
	VolumeMode   string   `json:"volumeMode"`
}

type AppMetadataHealthCheck struct {
}

type AppDeployOption struct {
	ZeroReplicas bool // If true, set replicas to 0 for initial deployment
	DebugMode    bool
}

func (a *AppMetadata) Deploy(ctx context.Context, cli client.Client, options *AppDeployOption) app.Error {
	if options != nil {
		if options.ZeroReplicas {
			a.Replicas = 0 // Set replicas to 0 for initial deployment
		}
		if options.DebugMode {
			a.ContainerCommand = "sleep infinity" // Set a debug command to keep the container running
		}
	}

	manifests, err := a.GetApplyManifests()
	if err != nil {
		return err
	}
	for _, resource := range manifests {
		if err := ApplyResource(ctx, cli, resource); err != nil {
			return err
		}
	}

	return nil
}

func (a *AppMetadata) Undeploy(ctx context.Context, cli client.Client) app.Error {
	manifests, err := a.GetApplyManifests()
	if err != nil {
		return err
	}
	for _, resource := range manifests {
		if err := DeleteResource(ctx, cli, resource); err != nil {
			return err
		}
	}

	return nil
}

func (a *AppMetadata) GetApplyManifests() ([]client.Object, app.Error) {
	switch a.WorkloadType {
	case app.WorkloadTypeDeployment:
		return a.deploymentManifests()
	case app.WorkloadTypeStatefulSet:
		return a.statefulSetManifests()
	default:
		return nil, app.NewError(http.StatusBadRequest, "Not supported workload type: "+a.WorkloadType)
	}
}

func (a *AppMetadata) standardSelectorLabels() map[string]string {
	return map[string]string{
		"ketches/owned":     "true",
		"ketches/app":       a.AppSlug,
		"ketches/env":       a.EnvSlug,
		"ketches/project":   a.ProjectSlug,
		"ketches/appID":     a.AppID,
		"ketches/envID":     a.EnvID,
		"ketches/projectID": a.ProjectID,
	}
}

func (a *AppMetadata) standardLabels() map[string]string {
	labels := a.standardSelectorLabels()
	labels["ketches/edition"] = a.Edition
	return labels
}

func (a *AppMetadata) deploymentManifests() ([]client.Object, app.Error) {
	var result []client.Object

	envs := make([]corev1.EnvVar, 0, len(a.EnvVars))
	for _, envVar := range a.EnvVars {
		envs = append(envs, corev1.EnvVar{
			Name:  envVar.Key,
			Value: envVar.Value,
		})
	}

	labels := a.standardLabels()
	selectorLabels := a.standardSelectorLabels()

	for _, pvc := range a.persistentVolumeClaimManifests() {
		result = append(result, &pvc)
	}
	volumeMounts := make([]corev1.VolumeMount, 0, len(a.Volumes))
	volumes := make([]corev1.Volume, 0, len(a.Volumes))
	for _, volume := range a.Volumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volume.Slug,
			MountPath: volume.MountPath,
			SubPath:   volume.SubPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: volume.Slug,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: volume.Slug,
				},
			},
		})
	}

	if len(a.Ports) > 0 {
		result = append(result, a.serviceManifest())
	}

	var (
		command []string
		args    []string
	)
	if a.ContainerCommand != "" {
		labels["ketches/debugging"] = "true" // Mark as debugging if command is set
		command = []string{"sh"}
		args = []string{"-c", a.ContainerCommand}
	}

	result = append(result, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.AppSlug,
			Namespace: a.ClusterNamespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &a.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            a.AppSlug,
							Image:           a.ContainerImage,
							ImagePullPolicy: corev1.PullAlways,
							Command:         command,
							Args:            args,
							Env:             envs,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", a.RequestCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", a.RequestMemory)),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", a.LimitCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", a.LimitMemory)),
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	})

	return result, nil
}

func (a *AppMetadata) statefulSetManifests() ([]client.Object, app.Error) {
	var result []client.Object

	envs := make([]corev1.EnvVar, 0, len(a.EnvVars))
	for _, envVar := range a.EnvVars {
		envs = append(envs, corev1.EnvVar{
			Name:  envVar.Key,
			Value: envVar.Value,
		})
	}

	labels := a.standardLabels()

	var (
		volumeClaims = a.persistentVolumeClaimManifests()
		volumeMounts = make([]corev1.VolumeMount, 0, len(a.Volumes))
		volumes      = make([]corev1.Volume, 0, len(a.Volumes))
	)
	for _, volume := range a.Volumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volume.Slug,
			MountPath: volume.MountPath,
			SubPath:   volume.SubPath,
		})

		volumes = append(volumes, corev1.Volume{
			Name: volume.Slug,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: volume.Slug,
				},
			},
		})
	}

	if len(a.Ports) > 0 {
		result = append(result, a.serviceManifest())
	}

	var (
		command []string
		args    []string
	)
	if a.ContainerCommand != "" {
		command = []string{"sh"}
		args = []string{"-c", a.ContainerCommand}
	}

	result = append(result, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.AppSlug,
			Namespace: a.ClusterNamespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			VolumeClaimTemplates: volumeClaims,
			ServiceName:          a.AppSlug,
			Replicas:             &a.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            a.AppSlug,
							Image:           a.ContainerImage,
							ImagePullPolicy: corev1.PullAlways,
							Command:         command,
							Args:            args,
							Env:             envs,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", a.RequestCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", a.RequestMemory)),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", a.LimitCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", a.LimitMemory)),
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			},
		},
	})

	return result, nil
}

func (a *AppMetadata) persistentVolumeClaimManifests() []corev1.PersistentVolumeClaim {
	var result []corev1.PersistentVolumeClaim

	for _, volume := range a.Volumes {
		result = append(result, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      volume.Slug,
				Namespace: a.ClusterNamespace,
				Labels:    a.standardLabels(),
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeMode:  utils.Ptr(corev1.PersistentVolumeMode(volume.VolumeMode)),
				AccessModes: pvcAccessModes(volume.AccessModes),
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dMi", volume.Capacity)),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dMi", volume.Capacity)),
					},
				},
				StorageClassName: &volume.StorageClass,
			},
		})
	}

	return result
}

func (a *AppMetadata) serviceManifest() *corev1.Service {
	labels := a.standardLabels()

	ports := make([]corev1.ServicePort, 0, len(a.Ports))
	for _, p := range a.Ports {
		if p.Protocol == "" {
			p.Protocol = string(corev1.ProtocolTCP)
		}
		if p.Port <= 0 {
			p.Port = 80 // Default port if not specified
		}
		ports = append(ports, corev1.ServicePort{
			Protocol: corev1.Protocol(p.Protocol),
			Port:     p.Port,
			TargetPort: intstr.IntOrString{
				IntVal: p.Port,
			},
		})
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.AppSlug,
			Namespace: a.ClusterNamespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    ports,
		},
	}
}

func pvcAccessModes(accessModes []string) []corev1.PersistentVolumeAccessMode {
	var kubeAccessModes []corev1.PersistentVolumeAccessMode
	for _, mode := range accessModes {
		kubeAccessModes = append(kubeAccessModes, corev1.PersistentVolumeAccessMode(mode))
	}
	return kubeAccessModes
}
