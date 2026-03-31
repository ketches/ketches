package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db/entities"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type builderWorkspaceExecutor interface {
	EnsureWorkspace(ctx context.Context, request builderWorkspaceExecutorRequest) (*builderExecutorSnapshot, error)
	GetWorkspaceSnapshot(ctx context.Context, handle *entities.BuilderExecutorHandle) (*builderExecutorSnapshot, error)
	CancelWorkspace(ctx context.Context, handle *entities.BuilderExecutorHandle) error
}

type builderWorkspaceExecutorRequest struct {
	Handle *entities.BuilderExecutorHandle

	SessionID      string
	ProjectID      string
	ProjectSlug    string
	BuildEnvID     string
	BuildEnvSlug   string
	ClusterID      string
	Namespace      string
	ExecutionImage string
	StorageRequest string
}

type builderExecutorSnapshot struct {
	HandleID      string
	Kind          entities.BuilderExecutorHandleKind
	Status        entities.BuilderExecutorHandleStatus
	ClusterID     string
	Namespace     string
	WorkloadName  string
	ContainerName string
}

func (snapshot *builderExecutorSnapshot) Usable() bool {
	return snapshot != nil && snapshot.Status == entities.BuilderExecutorHandleStatusActive
}

func (snapshot *builderExecutorSnapshot) Terminal() bool {
	if snapshot == nil {
		return false
	}

	switch snapshot.Status {
	case entities.BuilderExecutorHandleStatusFailed,
		entities.BuilderExecutorHandleStatusTerminating,
		entities.BuilderExecutorHandleStatusTerminated:
		return true
	default:
		return false
	}
}

var getBuilderWorkspaceExecutor = func(kind entities.BuilderExecutorHandleKind) (builderWorkspaceExecutor, error) {
	switch kind {
	case entities.BuilderExecutorHandleKindWorkspacePod:
		return builderWorkspacePodExecutorV1{}, nil
	default:
		return nil, fmt.Errorf("builder workspace executor kind %q is not supported", kind)
	}
}

type builderWorkspacePodExecutorV1 struct{}

func (builderWorkspacePodExecutorV1) EnsureWorkspace(ctx context.Context, request builderWorkspaceExecutorRequest) (*builderExecutorSnapshot, error) {
	if request.Handle == nil {
		return nil, errors.New("builder executor handle is required")
	}
	if request.Handle.ID == "" {
		return nil, errors.New("builder executor handle id is required")
	}
	if request.ClusterID == "" {
		return nil, errors.New("builder executor cluster id is required")
	}
	if request.Namespace == "" {
		return nil, errors.New("builder executor namespace is required")
	}

	resources, err := core.BuildBuilderWorkspaceResources(core.BuilderWorkspaceSpec{
		SessionID:      request.SessionID,
		ProjectID:      request.ProjectID,
		ProjectSlug:    request.ProjectSlug,
		BuildEnvID:     request.BuildEnvID,
		BuildEnvSlug:   request.BuildEnvSlug,
		Namespace:      request.Namespace,
		ExecutionImage: request.ExecutionImage,
		StorageRequest: request.StorageRequest,
	})
	if err != nil {
		return nil, err
	}

	client, err := getBuilderWorkspaceClusterClient(request.ClusterID)
	if err != nil {
		return nil, err
	}

	if _, err := client.CoreV1().PersistentVolumeClaims(request.Namespace).Create(ctx, resources.PersistentVolumeClaim, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}
	if _, err := client.CoreV1().Pods(request.Namespace).Create(ctx, resources.Pod, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}
	if err := waitForBuilderWorkspacePodReady(ctx, client, request.Namespace, resources.Pod.Name); err != nil {
		return nil, err
	}

	return &builderExecutorSnapshot{
		HandleID:      request.Handle.ID,
		Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
		Status:        entities.BuilderExecutorHandleStatusActive,
		ClusterID:     request.ClusterID,
		Namespace:     request.Namespace,
		WorkloadName:  resources.Pod.Name,
		ContainerName: resources.Pod.Spec.Containers[0].Name,
	}, nil
}

func (builderWorkspacePodExecutorV1) GetWorkspaceSnapshot(ctx context.Context, handle *entities.BuilderExecutorHandle) (*builderExecutorSnapshot, error) {
	clusterID, namespace, workloadName, containerName, err := builderWorkspaceExecutorHandleRuntimeRef(handle)
	if err != nil {
		return nil, err
	}

	client, err := getBuilderWorkspaceClusterClient(clusterID)
	if err != nil {
		return nil, err
	}

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, workloadName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &builderExecutorSnapshot{
				HandleID:      handle.ID,
				Kind:          handle.Kind,
				Status:        entities.BuilderExecutorHandleStatusTerminated,
				ClusterID:     clusterID,
				Namespace:     namespace,
				WorkloadName:  workloadName,
				ContainerName: containerName,
			}, nil
		}
		return nil, err
	}

	status := entities.BuilderExecutorHandleStatusProvisioning
	switch {
	case pod.DeletionTimestamp != nil:
		status = entities.BuilderExecutorHandleStatusTerminating
	case pod.Status.Phase == corev1.PodFailed:
		status = entities.BuilderExecutorHandleStatusFailed
	case pod.Status.Phase == corev1.PodSucceeded:
		status = entities.BuilderExecutorHandleStatusTerminated
	default:
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				status = entities.BuilderExecutorHandleStatusActive
				break
			}
		}
	}

	if len(pod.Spec.Containers) > 0 && pod.Spec.Containers[0].Name != "" {
		containerName = pod.Spec.Containers[0].Name
	}

	return &builderExecutorSnapshot{
		HandleID:      handle.ID,
		Kind:          handle.Kind,
		Status:        status,
		ClusterID:     clusterID,
		Namespace:     namespace,
		WorkloadName:  pod.Name,
		ContainerName: containerName,
	}, nil
}

func (builderWorkspacePodExecutorV1) CancelWorkspace(ctx context.Context, handle *entities.BuilderExecutorHandle) error {
	clusterID, namespace, workloadName, _, err := builderWorkspaceExecutorHandleRuntimeRef(handle)
	if err != nil {
		return err
	}

	client, err := getBuilderWorkspaceClusterClient(clusterID)
	if err != nil {
		return err
	}

	if err := client.CoreV1().Pods(namespace).Delete(ctx, workloadName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func reconcileBuilderWorkspaceFromExecutorSnapshot(workspace *entities.BuilderWorkspace, buildEnvID, workspaceRoot string, snapshot *builderExecutorSnapshot) error {
	if workspace == nil {
		return errors.New("builder workspace is required")
	}
	if snapshot == nil {
		return errors.New("builder executor snapshot is required")
	}
	if snapshot.HandleID == "" {
		return errors.New("builder executor snapshot handle id is required")
	}
	if snapshot.ClusterID == "" {
		return errors.New("builder executor snapshot cluster id is required")
	}
	if snapshot.Namespace == "" {
		return errors.New("builder executor snapshot namespace is required")
	}
	if snapshot.WorkloadName == "" {
		return errors.New("builder executor snapshot workload name is required")
	}
	if snapshot.ContainerName == "" {
		return errors.New("builder executor snapshot container name is required")
	}

	workspace.BuildEnvID = buildEnvID
	workspace.ExecutorHandleID = builderStringPtr(snapshot.HandleID)
	workspace.ClusterID = snapshot.ClusterID
	workspace.Namespace = snapshot.Namespace
	workspace.PodName = snapshot.WorkloadName
	workspace.ContainerName = snapshot.ContainerName
	workspace.Status = builderWorkspaceStatusFromExecutor(snapshot.Status)
	workspace.WorkspaceRoot = workspaceRoot

	return nil
}

func builderWorkspaceStatusFromExecutor(status entities.BuilderExecutorHandleStatus) entities.BuilderWorkspaceStatus {
	switch status {
	case entities.BuilderExecutorHandleStatusActive:
		return entities.BuilderWorkspaceStatusActive
	case entities.BuilderExecutorHandleStatusFailed,
		entities.BuilderExecutorHandleStatusTerminating,
		entities.BuilderExecutorHandleStatusTerminated:
		return entities.BuilderWorkspaceStatusFailed
	default:
		return entities.BuilderWorkspaceStatusProvisioning
	}
}

func builderWorkspaceExecutorHandleRuntimeRef(handle *entities.BuilderExecutorHandle) (clusterID, namespace, workloadName, containerName string, err error) {
	if handle == nil {
		return "", "", "", "", errors.New("builder executor handle is required")
	}
	if handle.ID == "" {
		return "", "", "", "", errors.New("builder executor handle id is required")
	}
	if handle.ClusterID == nil || *handle.ClusterID == "" {
		return "", "", "", "", errors.New("builder executor handle cluster id is required")
	}
	if handle.Namespace == nil || *handle.Namespace == "" {
		return "", "", "", "", errors.New("builder executor handle namespace is required")
	}
	if handle.WorkloadName == nil || *handle.WorkloadName == "" {
		return "", "", "", "", errors.New("builder executor handle workload name is required")
	}

	clusterID = *handle.ClusterID
	namespace = *handle.Namespace
	workloadName = *handle.WorkloadName
	if handle.ContainerName != nil {
		containerName = *handle.ContainerName
	}
	return clusterID, namespace, workloadName, containerName, nil
}

func builderStringPtr(value string) *string {
	return &value
}
