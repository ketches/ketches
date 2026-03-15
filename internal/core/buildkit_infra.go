package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	buildkitNamespaceName       = "ketches-build"
	buildkitServiceName         = "ketches-buildkitd"
	buildkitStatefulSetName     = "ketches-buildkitd"
	buildkitBinfmtDaemonSetName = "ketches-buildkit-binfmt"
	buildkitBuilderLabel        = "app.kubernetes.io/name=" + buildkitStatefulSetName
	buildkitBinfmtLabel         = "app.kubernetes.io/name=" + buildkitBinfmtDaemonSetName
	buildkitStateVolumeName     = "buildkit-state"
	buildkitPort                = int32(1234)
	buildkitdImage              = "moby/buildkit:buildx-stable-1"
	buildctlImage               = "moby/buildkit:buildx-stable-1"
	buildkitBinfmtImage         = "tonistiigi/binfmt:latest"
	buildkitPauseImage          = "registry.k8s.io/pause:3.10"
)

type BuildkitInfrastructureResources struct {
	Namespace       *corev1.Namespace
	Service         *corev1.Service
	StatefulSet     *appsv1.StatefulSet
	BinfmtDaemonSet *appsv1.DaemonSet
}

func buildkitInfrastructureResources() BuildkitInfrastructureResources {
	labels := map[string]string{
		"app.kubernetes.io/name":      buildkitServiceName,
		"app.kubernetes.io/component": "builder",
	}
	privileged := true
	replicas := int32(1)

	return BuildkitInfrastructureResources{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: buildkitNamespaceName,
			},
		},
		Service: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      buildkitServiceName,
				Namespace: buildkitNamespaceName,
				Labels:    labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Ports: []corev1.ServicePort{
					{
						Name: "buildkit",
						Port: buildkitPort,
					},
				},
			},
		},
		StatefulSet: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      buildkitStatefulSetName,
				Namespace: buildkitNamespaceName,
				Labels:    labels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: buildkitServiceName,
				Replicas:    &replicas,
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
								Name:  "buildkitd",
								Image: buildkitdImage,
								SecurityContext: &corev1.SecurityContext{
									Privileged: &privileged,
								},
								Args: []string{
									"--addr",
									"tcp://0.0.0.0:1234",
								},
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: buildkitPort,
										Name:          "buildkit",
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      buildkitStateVolumeName,
										MountPath: "/var/lib/buildkit",
									},
								},
							},
						},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: buildkitStateVolumeName,
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("50Gi"),
								},
							},
						},
					},
				},
			},
		},
		BinfmtDaemonSet: &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      buildkitBinfmtDaemonSetName,
				Namespace: buildkitNamespaceName,
				Labels: map[string]string{
					"app.kubernetes.io/name":      buildkitBinfmtDaemonSetName,
					"app.kubernetes.io/component": "builder",
				},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/name": buildkitBinfmtDaemonSetName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.kubernetes.io/name": buildkitBinfmtDaemonSetName,
						},
					},
					Spec: corev1.PodSpec{
						HostPID:       true,
						RestartPolicy: corev1.RestartPolicyAlways,
						InitContainers: []corev1.Container{
							{
								Name:  "install-binfmt",
								Image: buildkitBinfmtImage,
								Args:  []string{"--install", "all"},
								SecurityContext: &corev1.SecurityContext{
									Privileged: &privileged,
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "pause",
								Image: buildkitPauseImage,
							},
						},
					},
				},
			},
		},
	}
}

func EnsureClusterBuildkitInfrastructure(ctx context.Context, clusterID string) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	resources := buildkitInfrastructureResources()

	if _, err := client.CoreV1().Namespaces().Get(ctx, resources.Namespace.Name, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			if _, err := client.CoreV1().Namespaces().Create(ctx, resources.Namespace, metav1.CreateOptions{}); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if err := applyBuildkitService(ctx, client.CoreV1().Services(buildkitNamespaceName), resources.Service); err != nil {
		return err
	}
	if err := applyBuildkitStatefulSet(ctx, client.AppsV1().StatefulSets(buildkitNamespaceName), resources.StatefulSet); err != nil {
		return err
	}
	if err := applyBuildkitDaemonSet(ctx, client.AppsV1().DaemonSets(buildkitNamespaceName), resources.BinfmtDaemonSet); err != nil {
		return err
	}

	return nil
}

func EnsureBuildkitBuildPrerequisites(ctx context.Context, clusterID, platforms string) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	statefulSet, err := client.AppsV1().StatefulSets(buildkitNamespaceName).Get(ctx, buildkitStatefulSetName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if err := validateBuildkitStatefulSetReadiness(statefulSet); err != nil {
		return err
	}

	builderPods, err := client.CoreV1().Pods(buildkitNamespaceName).List(ctx, metav1.ListOptions{
		LabelSelector: buildkitBuilderLabel,
	})
	if err != nil {
		return err
	}

	builderNodeName, err := buildkitNodeName(builderPods.Items)
	if err != nil {
		return err
	}

	if !buildkitRequiresBinfmt(platforms) {
		return nil
	}

	binfmtPods, err := client.CoreV1().Pods(buildkitNamespaceName).List(ctx, metav1.ListOptions{
		LabelSelector: buildkitBinfmtLabel,
	})
	if err != nil {
		return err
	}
	return validateBinfmtPodReadinessOnNode(platforms, builderNodeName, binfmtPods.Items)
}

func buildkitRequiresBinfmt(platforms string) bool {
	return strings.Contains(strings.TrimSpace(platforms), ",")
}

func validateBuildkitStatefulSetReadiness(sts *appsv1.StatefulSet) error {
	if sts == nil {
		return fmt.Errorf("BuildKit builder is not ready in namespace %s. Wait for StatefulSet %s to become Ready before retrying.", buildkitNamespaceName, buildkitStatefulSetName)
	}

	desiredReplicas := int32(1)
	if sts.Spec.Replicas != nil && *sts.Spec.Replicas > 0 {
		desiredReplicas = *sts.Spec.Replicas
	}
	if sts.Status.ReadyReplicas < desiredReplicas {
		return fmt.Errorf("BuildKit builder is not ready in namespace %s. Wait for StatefulSet %s to become Ready before retrying.", buildkitNamespaceName, buildkitStatefulSetName)
	}
	return nil
}

func validateBinfmtDaemonSetReadiness(platforms string, ds *appsv1.DaemonSet) error {
	if !buildkitRequiresBinfmt(platforms) {
		return nil
	}
	if ds == nil || ds.Status.DesiredNumberScheduled == 0 || ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
		return fmt.Errorf("Multi-arch build requires binfmt/QEMU support, but DaemonSet %s is not ready in namespace %s.", buildkitBinfmtDaemonSetName, buildkitNamespaceName)
	}
	return nil
}

func buildkitNodeName(pods []corev1.Pod) (string, error) {
	for _, pod := range pods {
		if podReady(&pod) && strings.TrimSpace(pod.Spec.NodeName) != "" {
			return pod.Spec.NodeName, nil
		}
	}

	return "", fmt.Errorf("BuildKit builder Pod is not ready in namespace %s. Wait for Pod from StatefulSet %s to become Ready before retrying.", buildkitNamespaceName, buildkitStatefulSetName)
}

func validateBinfmtPodReadinessOnNode(platforms, nodeName string, pods []corev1.Pod) error {
	if !buildkitRequiresBinfmt(platforms) {
		return nil
	}

	for _, pod := range pods {
		if pod.Spec.NodeName != nodeName {
			continue
		}
		if podReady(&pod) {
			return nil
		}
	}

	return fmt.Errorf("Multi-arch build requires binfmt/QEMU support on builder node %s, but Pod for DaemonSet %s is not Ready in namespace %s.", nodeName, buildkitBinfmtDaemonSetName, buildkitNamespaceName)
}

func podReady(pod *corev1.Pod) bool {
	if pod == nil || pod.DeletionTimestamp != nil || pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

func applyBuildkitService(ctx context.Context, client interface {
	Get(context.Context, string, metav1.GetOptions) (*corev1.Service, error)
	Create(context.Context, *corev1.Service, metav1.CreateOptions) (*corev1.Service, error)
	Update(context.Context, *corev1.Service, metav1.UpdateOptions) (*corev1.Service, error)
}, svc *corev1.Service) error {
	existing, err := client.Get(ctx, svc.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, svc, metav1.CreateOptions{})
			return err
		}
		return err
	}

	svc.ResourceVersion = existing.ResourceVersion
	svc.Spec.ClusterIP = existing.Spec.ClusterIP
	svc.Spec.ClusterIPs = existing.Spec.ClusterIPs
	svc.Spec.IPFamilies = existing.Spec.IPFamilies
	svc.Spec.IPFamilyPolicy = existing.Spec.IPFamilyPolicy
	svc.Spec.HealthCheckNodePort = existing.Spec.HealthCheckNodePort

	_, err = client.Update(ctx, svc, metav1.UpdateOptions{})
	return err
}

func applyBuildkitStatefulSet(ctx context.Context, client interface {
	Get(context.Context, string, metav1.GetOptions) (*appsv1.StatefulSet, error)
	Create(context.Context, *appsv1.StatefulSet, metav1.CreateOptions) (*appsv1.StatefulSet, error)
	Update(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) (*appsv1.StatefulSet, error)
}, sts *appsv1.StatefulSet) error {
	existing, err := client.Get(ctx, sts.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, sts, metav1.CreateOptions{})
			return err
		}
		return err
	}

	sts.ResourceVersion = existing.ResourceVersion
	_, err = client.Update(ctx, sts, metav1.UpdateOptions{})
	return err
}

func applyBuildkitDaemonSet(ctx context.Context, client interface {
	Get(context.Context, string, metav1.GetOptions) (*appsv1.DaemonSet, error)
	Create(context.Context, *appsv1.DaemonSet, metav1.CreateOptions) (*appsv1.DaemonSet, error)
	Update(context.Context, *appsv1.DaemonSet, metav1.UpdateOptions) (*appsv1.DaemonSet, error)
}, ds *appsv1.DaemonSet) error {
	existing, err := client.Get(ctx, ds.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, ds, metav1.CreateOptions{})
			return err
		}
		return err
	}

	ds.ResourceVersion = existing.ResourceVersion
	_, err = client.Update(ctx, ds, metav1.UpdateOptions{})
	return err
}
