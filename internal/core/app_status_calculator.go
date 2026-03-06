package core

import (
	"context"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CalculateAppStatus(ctx context.Context, application *entities.App) (app.AppStatus, error) {
	// If not deployed yet, return undeployed directly
	if application.DeployStatus == "undeployed" {
		return app.AppStatusUndeployed, nil
	}

	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return app.AppStatusUnknown, err
	}

	namespace := application.Env.ClusterNamespace
	appName := application.Slug

	// Check if any pod has debugging label
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.ketches.cn/slug=" + appName,
	})
	if err != nil {
		return app.AppStatusUnknown, err
	}

	isDebugging := false
	for _, pod := range pods.Items {
		if pod.Labels["app.ketches.cn/debugging"] == "true" {
			isDebugging = true
			break
		}
	}

	switch application.AppType {
	case "Deployment":
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, appName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return app.AppStatusUndeployed, nil
			}
			return app.AppStatusUnknown, err
		}

		if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas == 0 {
			if deployment.Status.Replicas == 0 {
				return app.AppStatusStopped, nil
			}
			return app.AppStatusStopping, nil
		}

		if deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
			return app.AppStatusUpdating, nil
		}

		if deployment.Status.ReadyReplicas == 0 {
			return app.AppStatusStarting, nil
		}

		if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
			return app.AppStatusAbnormal, nil
		}

		if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas {
			if isDebugging {
				return app.AppStatusDebugging, nil
			}
			return app.AppStatusRunning, nil
		}

		return app.AppStatusAbnormal, nil

	case "StatefulSet":
		statefulSet, err := client.AppsV1().StatefulSets(namespace).Get(ctx, appName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return app.AppStatusUndeployed, nil
			}
			return app.AppStatusUnknown, err
		}

		if statefulSet.Spec.Replicas == nil || *statefulSet.Spec.Replicas == 0 {
			if statefulSet.Status.Replicas == 0 {
				return app.AppStatusStopped, nil
			}
			return app.AppStatusStopping, nil
		}

		if statefulSet.Status.UpdatedReplicas < *statefulSet.Spec.Replicas {
			return app.AppStatusUpdating, nil
		}

		if statefulSet.Status.ReadyReplicas == 0 {
			return app.AppStatusStarting, nil
		}

		if statefulSet.Status.ReadyReplicas < *statefulSet.Spec.Replicas {
			return app.AppStatusAbnormal, nil
		}

		if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
			if isDebugging {
				return app.AppStatusDebugging, nil
			}
			return app.AppStatusRunning, nil
		}

		return app.AppStatusAbnormal, nil
	}

	return app.AppStatusUnknown, nil
}

// CalculateAppListStatus computes the live status for an app from flat listing
// parameters (no entity associations required). Used by the listing DTO path.
func CalculateAppListStatus(ctx context.Context, client kubernetes.Interface, appID, appSlug, appType, namespace string, replicas int) (app.AppStatus, error) {
	// Check if any pod has debugging label
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.ketches.cn/slug=" + appSlug,
	})
	if err != nil {
		return app.AppStatusUnknown, err
	}

	isDebugging := false
	for _, pod := range pods.Items {
		if pod.Labels["app.ketches.cn/debugging"] == "true" {
			isDebugging = true
			break
		}
	}

	switch appType {
	case "Deployment":
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, appSlug, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return app.AppStatusUndeployed, nil
			}
			return app.AppStatusUnknown, err
		}

		if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas == 0 {
			if deployment.Status.Replicas == 0 {
				return app.AppStatusStopped, nil
			}
			return app.AppStatusStopping, nil
		}

		if deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
			return app.AppStatusUpdating, nil
		}

		if deployment.Status.ReadyReplicas == 0 {
			return app.AppStatusStarting, nil
		}

		if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
			return app.AppStatusAbnormal, nil
		}

		if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas {
			if isDebugging {
				return app.AppStatusDebugging, nil
			}
			return app.AppStatusRunning, nil
		}

		return app.AppStatusAbnormal, nil

	case "StatefulSet":
		statefulSet, err := client.AppsV1().StatefulSets(namespace).Get(ctx, appSlug, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return app.AppStatusUndeployed, nil
			}
			return app.AppStatusUnknown, err
		}

		if statefulSet.Spec.Replicas == nil || *statefulSet.Spec.Replicas == 0 {
			if statefulSet.Status.Replicas == 0 {
				return app.AppStatusStopped, nil
			}
			return app.AppStatusStopping, nil
		}

		if statefulSet.Status.UpdatedReplicas < *statefulSet.Spec.Replicas {
			return app.AppStatusUpdating, nil
		}

		if statefulSet.Status.ReadyReplicas == 0 {
			return app.AppStatusStarting, nil
		}

		if statefulSet.Status.ReadyReplicas < *statefulSet.Spec.Replicas {
			return app.AppStatusAbnormal, nil
		}

		if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
			if isDebugging {
				return app.AppStatusDebugging, nil
			}
			return app.AppStatusRunning, nil
		}

		return app.AppStatusAbnormal, nil
	}

	return app.AppStatusUnknown, nil
}
