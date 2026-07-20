package services

import (
	"context"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// errPodAccessDenied is kept separate from Kubernetes errors so callers can
// distinguish an authorization failure from a cluster or transport failure.
var errPodAccessDenied = app.NewErrorf("pod access denied")

// validateAppPodAccess reads the requested Pod and verifies that it belongs to
// the application or to an explicitly scoped internal workload.
func validateAppPodAccess(ctx context.Context, client kubernetes.Interface, appCtx *models.AppContext, instanceName string) (*corev1.Pod, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		return nil, app.NewErrorf("kubernetes client is required")
	}
	if appCtx == nil {
		return nil, app.NewErrorf("application context is required")
	}
	if strings.TrimSpace(instanceName) == "" {
		return nil, app.NewErrorf("pod name is required")
	}

	namespace := strings.TrimSpace(appCtx.EnvContext.Env.ClusterNamespace)
	if namespace == "" {
		return nil, app.NewErrorf("cluster namespace is required")
	}

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, instanceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if pod.Labels[kube.LabelManagedBy] != "true" {
		return nil, app.WrapErrorf(errPodAccessDenied, "pod %q is not managed by Ketches", instanceName)
	}

	if appCtx.App.ID != "" {
		// Build Pods can carry the application ID because they are created for
		// an application build. They are never valid application instances.
		if pod.Labels[kube.LabelBuildKey] == "true" || pod.Labels[kube.LabelBuildID] != "" || pod.Labels[kube.LabelAppID] != appCtx.App.ID {
			return nil, app.WrapErrorf(errPodAccessDenied, "pod %q does not belong to application", instanceName)
		}
		return pod, nil
	}

	if appCtx.PodAccessPolicy == nil || len(appCtx.PodAccessPolicy.RequiredLabels) == 0 {
		return nil, app.NewErrorf("application ID or Pod access policy is required")
	}
	for key, value := range appCtx.PodAccessPolicy.RequiredLabels {
		if pod.Labels[key] != value {
			return nil, app.WrapErrorf(errPodAccessDenied, "pod %q does not match its access policy", instanceName)
		}
	}

	return pod, nil
}

func validateAppPodContainerAccess(pod *corev1.Pod, containerName string) error {
	if pod == nil {
		return app.NewErrorf("pod is required")
	}
	containerName = strings.TrimSpace(containerName)
	if containerName == "" {
		return app.WrapErrorf(errPodAccessDenied, "container name is required")
	}

	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			return nil
		}
	}
	for _, container := range pod.Spec.InitContainers {
		if container.Name == containerName {
			return nil
		}
	}

	return app.WrapErrorf(errPodAccessDenied, "container %q is not allowed for pod %q", containerName, pod.Name)
}

func validateAppPodContainer(ctx context.Context, client kubernetes.Interface, appCtx *models.AppContext, instanceName, containerName string) (*corev1.Pod, error) {
	pod, err := validateAppPodAccess(ctx, client, appCtx, instanceName)
	if err != nil {
		return nil, err
	}
	if err := validateAppPodContainerAccess(pod, containerName); err != nil {
		return nil, err
	}
	return pod, nil
}
