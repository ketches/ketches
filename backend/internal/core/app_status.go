package core

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	corev1 "k8s.io/api/core/v1"
)

type AppWorkloadStatus = string

const (
	AppWorkloadStatusUndeployed AppWorkloadStatus = "undeployed"
	AppWorkloadStatusStarting   AppWorkloadStatus = "starting"
	AppWorkloadStatusRunning    AppWorkloadStatus = "running"
	AppWorkloadStatusStopped    AppWorkloadStatus = "stopped"
	AppWorkloadStatusStopping   AppWorkloadStatus = "stopping"
	AppWorkloadStatusUpdating   AppWorkloadStatus = "updating"
	AppWorkloadStatusAbnormal   AppWorkloadStatus = "abnormal"
	AppWorkloadStatusCompleted  AppWorkloadStatus = "completed"
	AppWorkloadStatusDebugging  AppWorkloadStatus = "debugging"
	AppWorkloadStatusUnknown    AppWorkloadStatus = "unknown"
)

type AppStatus struct {
	DesiredReplicas int32             `json:"desiredReplicas"`
	DesiredEdition  string            `json:"desiredEdition"`
	ActualReplicas  int32             `json:"actualReplicas"`
	ActualEdition   string            `json:"actualEdition"`
	Status          AppWorkloadStatus `json:"status"`
}

func GetAppStatus(ctx context.Context, appEntity *entities.App) AppStatus {
	result := AppStatus{
		DesiredReplicas: appEntity.Replicas,
		DesiredEdition:  appEntity.Edition,
	}
	switch appEntity.WorkloadType {
	case app.WorkloadTypeDeployment:
		deployment, err := kube.GetDeployment(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
		if err != nil {
			result.ActualReplicas = 0
			result.ActualEdition = ""
			if err.Code() == http.StatusNotFound {
				result.Status = AppWorkloadStatusUndeployed
				return result
			}
			result.Status = AppWorkloadStatusUnknown
			return result
		}
		result.ActualEdition = deployment.Labels["ketches/edition"]
		if deployment.Labels["ketches/debugging"] == "true" {
			result.Status = AppWorkloadStatusDebugging
			return result
		}
	case app.WorkloadTypeStatefulSet:
		statefulSet, err := kube.GetStatefulSet(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
		if err != nil {
			if err.Code() == http.StatusNotFound {
				result.Status = AppWorkloadStatusUndeployed
				return result
			}
			result.Status = AppWorkloadStatusUnknown
			return result
		}
		result.ActualEdition = statefulSet.Labels["ketches/edition"]
		if statefulSet.Labels["ketches/debugging"] == "true" {
			result.Status = AppWorkloadStatusDebugging
			return result
		}
	}

	pods, err := kube.ListPods(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
	if err != nil {
		result.Status = AppWorkloadStatusUnknown
		return result
	}

	if len(pods) == 0 {
		result.Status = AppWorkloadStatusStopped
		return result
	}

	result.ActualReplicas = int32(len(pods))

	var (
		updating            bool
		runningPodCount     int32
		pendingPodCount     int32
		terminatingPodCount int32
	)
	for _, pod := range pods {
		if kube.IsPodAbnormal(pod) {
			result.Status = AppWorkloadStatusAbnormal
		}

		edition := pod.Labels["ketches/edition"]
		if edition != result.ActualEdition {
			updating = true
			continue
		}

		if pod.DeletionTimestamp != nil {
			terminatingPodCount++
			continue
		}

		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningPodCount++
		case corev1.PodPending:
			pendingPodCount++
		}
	}

	if updating {
		result.Status = AppWorkloadStatusUpdating
		return result
	}

	if terminatingPodCount > 0 {
		if runningPodCount == 0 && pendingPodCount == 0 {
			result.Status = AppWorkloadStatusStopped
			return result
		} else {
			result.Status = AppWorkloadStatusStopping
			return result
		}
	}

	if pendingPodCount > 0 {
		result.Status = AppWorkloadStatusStarting
		return result
	}

	if runningPodCount == result.ActualReplicas {
		result.Status = AppWorkloadStatusRunning
		return result
	}

	result.Status = AppWorkloadStatusUnknown
	return result
}
