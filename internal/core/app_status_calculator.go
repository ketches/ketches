package core

import (
	"context"
	"sort"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CalculateAppStatus(ctx context.Context, appCtx *models.AppContext) (app.AppStatus, error) {
	// If not deployed yet, return undeployed directly
	if appCtx.App.DeployStatus == app.AppStatusUndeployed {
		return app.AppStatusUndeployed, nil
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return app.AppStatusUnknown, err
	}

	namespace := appCtx.EnvContext.Env.ClusterNamespace
	appName := appCtx.App.Slug

	// Check if any pod has debugging label
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: kube.LabelAppSlug + "=" + appName,
	})
	if err != nil {
		return app.AppStatusUnknown, err
	}

	isDebugging := hasDebuggingPod(pods.Items)

	switch appCtx.App.AppType {
	case app.AppTypeDeployment:
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, appName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return app.AppStatusUndeployed, nil
			}
			return app.AppStatusUnknown, err
		}
		return calculateDeploymentStatus(deployment, isDebugging), nil

	case app.AppTypeStatefulSet:
		statefulSet, err := client.AppsV1().StatefulSets(namespace).Get(ctx, appName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return app.AppStatusUndeployed, nil
			}
			return app.AppStatusUnknown, err
		}
		return calculateStatefulSetStatus(statefulSet, isDebugging), nil
	}

	return app.AppStatusUnknown, nil
}

// CalculateAppListStatuses computes live statuses for multiple apps in the
// same cluster namespace using batched K8s list calls.
func CalculateAppListStatuses(ctx context.Context, client kubernetes.Interface, namespace string, rows []models.AppListRow) (map[string]app.AppStatus, error) {
	statuses := make(map[string]app.AppStatus, len(rows))
	if len(rows) == 0 {
		return statuses, nil
	}

	selector := buildManagedAppSlugSelector(rows)

	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	deploymentList, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	statefulSetList, err := client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	podsBySlug := groupPodsBySlug(podList.Items)
	deploymentsBySlug := mapDeploymentsBySlug(deploymentList.Items)
	statefulSetsBySlug := mapStatefulSetsBySlug(statefulSetList.Items)

	for i := range rows {
		row := rows[i]
		isDebugging := hasDebuggingPod(podsBySlug[row.Slug])

		switch row.AppType {
		case app.AppTypeDeployment:
			statuses[row.ID] = calculateDeploymentStatus(deploymentsBySlug[row.Slug], isDebugging)
		case app.AppTypeStatefulSet:
			statuses[row.ID] = calculateStatefulSetStatus(statefulSetsBySlug[row.Slug], isDebugging)
		default:
			statuses[row.ID] = app.AppStatusUnknown
		}
	}

	return statuses, nil
}

// CalculateAppListStatus computes the live status for an app from flat listing
// parameters (no entity associations required). Used by the listing DTO path.
func CalculateAppListStatus(ctx context.Context, client kubernetes.Interface, appID, appSlug, appType, namespace string, replicas int) (app.AppStatus, error) {
	// Check if any pod has debugging label
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: kube.LabelAppSlug + "=" + appSlug,
	})
	if err != nil {
		return app.AppStatusUnknown, err
	}

	isDebugging := hasDebuggingPod(pods.Items)

	switch appType {
	case "Deployment":
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, appSlug, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return app.AppStatusUndeployed, nil
			}
			return app.AppStatusUnknown, err
		}
		return calculateDeploymentStatus(deployment, isDebugging), nil

	case app.AppTypeStatefulSet:
		statefulSet, err := client.AppsV1().StatefulSets(namespace).Get(ctx, appSlug, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return app.AppStatusUndeployed, nil
			}
			return app.AppStatusUnknown, err
		}
		return calculateStatefulSetStatus(statefulSet, isDebugging), nil
	}

	return app.AppStatusUnknown, nil
}

func buildManagedAppSlugSelector(rows []models.AppListRow) string {
	slugSet := make(map[string]struct{}, len(rows))
	for i := range rows {
		slug := strings.TrimSpace(rows[i].Slug)
		if slug == "" {
			continue
		}
		slugSet[slug] = struct{}{}
	}

	slugs := make([]string, 0, len(slugSet))
	for slug := range slugSet {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	managedSelector := kube.LabelManagedBy + "=true"
	if len(slugs) == 0 {
		return managedSelector
	}

	return managedSelector + "," + kube.LabelAppSlug + " in (" + strings.Join(slugs, ",") + ")"
}

func groupPodsBySlug(pods []corev1.Pod) map[string][]corev1.Pod {
	result := make(map[string][]corev1.Pod)
	for i := range pods {
		slug := objectAppSlug(pods[i].Labels, pods[i].Name)
		if slug == "" {
			continue
		}
		result[slug] = append(result[slug], pods[i])
	}
	return result
}

func mapDeploymentsBySlug(deployments []appsv1.Deployment) map[string]*appsv1.Deployment {
	result := make(map[string]*appsv1.Deployment, len(deployments))
	for i := range deployments {
		deployment := deployments[i]
		slug := objectAppSlug(deployment.Labels, deployment.Name)
		if slug == "" {
			continue
		}
		result[slug] = &deployment
	}
	return result
}

func mapStatefulSetsBySlug(statefulSets []appsv1.StatefulSet) map[string]*appsv1.StatefulSet {
	result := make(map[string]*appsv1.StatefulSet, len(statefulSets))
	for i := range statefulSets {
		statefulSet := statefulSets[i]
		slug := objectAppSlug(statefulSet.Labels, statefulSet.Name)
		if slug == "" {
			continue
		}
		result[slug] = &statefulSet
	}
	return result
}

func objectAppSlug(labels map[string]string, fallbackName string) string {
	if labels != nil {
		if slug := strings.TrimSpace(labels[kube.LabelAppSlug]); slug != "" {
			return slug
		}
	}
	return strings.TrimSpace(fallbackName)
}

func hasDebuggingPod(pods []corev1.Pod) bool {
	for i := range pods {
		if pods[i].Labels[kube.LabelDebugging] == "true" {
			return true
		}
	}
	return false
}

func calculateDeploymentStatus(deployment *appsv1.Deployment, isDebugging bool) app.AppStatus {
	if deployment == nil {
		return app.AppStatusUndeployed
	}

	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas == 0 {
		if deployment.Status.Replicas == 0 {
			return app.AppStatusStopped
		}
		return app.AppStatusStopping
	}

	if deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
		return app.AppStatusUpdating
	}

	if deployment.Status.ReadyReplicas == 0 {
		return app.AppStatusStarting
	}

	if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
		return app.AppStatusAbnormal
	}

	if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas {
		if isDebugging {
			return app.AppStatusDebugging
		}
		return app.AppStatusRunning
	}

	return app.AppStatusAbnormal
}

func calculateStatefulSetStatus(statefulSet *appsv1.StatefulSet, isDebugging bool) app.AppStatus {
	if statefulSet == nil {
		return app.AppStatusUndeployed
	}

	if statefulSet.Spec.Replicas == nil || *statefulSet.Spec.Replicas == 0 {
		if statefulSet.Status.Replicas == 0 {
			return app.AppStatusStopped
		}
		return app.AppStatusStopping
	}

	if statefulSet.Status.UpdatedReplicas < *statefulSet.Spec.Replicas {
		return app.AppStatusUpdating
	}

	if statefulSet.Status.ReadyReplicas == 0 {
		return app.AppStatusStarting
	}

	if statefulSet.Status.ReadyReplicas < *statefulSet.Spec.Replicas {
		return app.AppStatusAbnormal
	}

	if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
		if isDebugging {
			return app.AppStatusDebugging
		}
		return app.AppStatusRunning
	}

	return app.AppStatusAbnormal
}
