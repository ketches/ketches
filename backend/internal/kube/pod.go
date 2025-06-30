package kube

import (
	"context"
	"log"
	"time"

	"github.com/ketches/ketches/internal/app"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func ListPods(ctx context.Context, clusterID, namespace, appSlug string) ([]*corev1.Pod, app.Error) {
	store, e := ClusterStore(ctx, clusterID)
	if e != nil {
		return nil, e
	}

	pods, err := store.PodLister().Pods(namespace).List(labels.SelectorFromSet(labels.Set{
		"ketches/app": appSlug,
	}))
	if err != nil {
		log.Printf("Failed to list pods in cluster %s, namespace %s, app %s: %v", clusterID, namespace, appSlug, err)
		return nil, app.ErrClusterOperationFailed
	}

	return pods, nil
}

func DeletePod(ctx context.Context, clusterID, namespace, podName string) app.Error {
	clientset, e := ClusterClientset(ctx, clusterID, false)
	if e != nil {
		return e
	}

	if err := clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{}); err != nil {
		log.Printf("Failed to delete pod %s in namespace %s of cluster %s: %v", podName, namespace, clusterID, err)
		return app.ErrClusterOperationFailed
	}

	return nil
}

type PodStatus string

const (
	PodStatusRunning     PodStatus = "running"
	PodStatusSucceeded   PodStatus = "succeeded"
	PodStatusPending     PodStatus = "pending"
	PodStatusTerminating PodStatus = "terminating"
	PodStatusAbnormal    PodStatus = "abnormal"
	PodStatusDebugging   PodStatus = "debugging"
	PodStatusUnknown     PodStatus = "unknown"
)

func GetPodStatus(pod *corev1.Pod) string {
	if pod == nil {
		return string(PodStatusUnknown)
	}

	if pod.Labels["ketches/debugging"] == "true" {
		return string(PodStatusDebugging)
	}

	if IsAbnormalPod(pod) {
		return string(PodStatusAbnormal)
	}

	switch pod.Status.Phase {
	case corev1.PodRunning:
		return string(PodStatusRunning)
	case corev1.PodSucceeded:
		return string(PodStatusSucceeded)
	case corev1.PodPending:
		return string(PodStatusPending)
	case corev1.PodFailed:
		return string(PodStatusAbnormal)
	default:
		return string(PodStatusPending)
	}
}

func GetContainerStatus(cs *corev1.ContainerStatus) string {
	if cs == nil {
		return "Unknown"
	}

	if cs.State.Running != nil {
		return "Running"
	}
	if cs.State.Waiting != nil {
		return "Waiting: " + cs.State.Waiting.Reason
	}
	if cs.State.Terminated != nil {
		return "Terminated: " + cs.State.Terminated.Reason
	}
	return "Unknown"
}

func IsAbnormalPod(pod *corev1.Pod) bool {
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
		return true
	}

	now := time.Now()
	for _, cond := range pod.Status.Conditions {
		switch cond.Type {
		case corev1.PodReady, corev1.ContainersReady:
			if cond.Status != corev1.ConditionTrue && cond.LastTransitionTime.Time.Before(now.Add(-2*time.Minute)) {
				return true
			}
		case corev1.PodScheduled:
			if cond.Status != corev1.ConditionTrue && cond.Reason == "Unschedulable" {
				return true
			}
		}
	}

	const restartThreshold = 3
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			switch cs.State.Waiting.Reason {
			case "CrashLoopBackOff",
				"ImagePullBackOff",
				"ErrImagePull",
				"RunContainerError":
				return true
			case "CreateContainerConfigError",
				"SetupContainerError",
				"AttachVolume.Attach failed",
				"MountVolume.SetUp failed":
				return true
			}
		}

		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			return true
		}

		if cs.RestartCount > restartThreshold {
			return true
		}
	}

	return false
}

func MainContainer(appSlug string, pod *corev1.Pod) *corev1.Container {
	for _, container := range pod.Spec.Containers {
		if container.Name == appSlug {
			return &container
		}
	}
	return nil
}
