package core

import (
	"time"

	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
)

func ToAppInstanceResponse(pod *corev1.Pod) models.AppInstanceResponse {
	duration := "-"
	if pod.Status.StartTime != nil {
		d := time.Since(pod.Status.StartTime.Time).Round(time.Second)
		duration = d.String()
	}

	var containerNames []string
	for _, container := range pod.Spec.Containers {
		containerNames = append(containerNames, container.Name)
	}

	var initContainerNames []string
	for _, container := range pod.Spec.InitContainers {
		initContainerNames = append(initContainerNames, container.Name)
	}

	restartCount := 0
	for _, cs := range pod.Status.ContainerStatuses {
		restartCount += int(cs.RestartCount)
	}
	for _, cs := range pod.Status.InitContainerStatuses {
		restartCount += int(cs.RestartCount)
	}

	return models.AppInstanceResponse{
		InstanceName:       pod.Name,
		Status:             string(pod.Status.Phase),
		IP:                 pod.Status.PodIP,
		InitContainerCount: len(pod.Spec.InitContainers),
		InitContainers:     initContainerNames,
		ContainerCount:     len(pod.Spec.Containers),
		Containers:         containerNames,
		NodeName:           pod.Spec.NodeName,
		NodeIP:             pod.Status.HostIP,
		RestartCount:       restartCount,
		RunningDuration:    duration,
		CreatedAt:          pod.CreationTimestamp.Time,
	}
}
