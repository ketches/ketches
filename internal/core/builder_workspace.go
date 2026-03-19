package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	builderWorkspaceMarkerLabelKey = kube.LabelPrefix + "builder-workspace"
	builderSessionIDLabelKey       = kube.LabelPrefix + "builder-session-id"
	builderWorkspaceContainerName  = "workspace"
	builderWorkspaceVolumeName     = "workspace"
)

type BuilderWorkspaceSpec struct {
	SessionID      string
	ProjectID      string
	ProjectSlug    string
	BuildEnvID     string
	BuildEnvSlug   string
	Namespace      string
	StorageRequest string
}

type BuilderWorkspaceResources struct {
	PersistentVolumeClaim *corev1.PersistentVolumeClaim
	Pod                   *corev1.Pod
	Service               *corev1.Service
}

func BuildBuilderWorkspaceResources(spec BuilderWorkspaceSpec) (BuilderWorkspaceResources, error) {
	pvc, err := BuildBuilderWorkspacePVC(spec)
	if err != nil {
		return BuilderWorkspaceResources{}, err
	}

	pod, err := BuildBuilderWorkspacePod(spec)
	if err != nil {
		return BuilderWorkspaceResources{}, err
	}

	return BuilderWorkspaceResources{
		PersistentVolumeClaim: pvc,
		Pod:                   pod,
	}, nil
}

func BuildBuilderWorkspacePVC(spec BuilderWorkspaceSpec) (*corev1.PersistentVolumeClaim, error) {
	if err := validateBuilderWorkspaceStorageSpec(spec); err != nil {
		return nil, err
	}

	storageRequest, err := resource.ParseQuantity(strings.TrimSpace(spec.StorageRequest))
	if err != nil {
		return nil, fmt.Errorf("builder workspace storage request %q is invalid: %w", spec.StorageRequest, err)
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builderWorkspacePVCName(spec.SessionID),
			Namespace: spec.Namespace,
			Labels:    builderWorkspaceLabels(spec),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageRequest,
				},
			},
		},
	}, nil
}

func BuildBuilderWorkspacePod(spec BuilderWorkspaceSpec) (*corev1.Pod, error) {
	if err := validateBuilderWorkspacePodSpec(spec); err != nil {
		return nil, err
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builderWorkspacePodName(spec.SessionID),
			Namespace: spec.Namespace,
			Labels:    builderWorkspaceLabels(spec),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  builderWorkspaceContainerName,
					Image: app.Config.BuilderWorkspaceImage,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      builderWorkspaceVolumeName,
							MountPath: app.Config.BuilderWorkspaceRoot,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: builderWorkspaceVolumeName,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: builderWorkspacePVCName(spec.SessionID),
						},
					},
				},
			},
		},
	}, nil
}

func builderWorkspaceLabels(spec BuilderWorkspaceSpec) map[string]string {
	return map[string]string{
		kube.LabelProjectID:            spec.ProjectID,
		kube.LabelProjectSlug:          spec.ProjectSlug,
		kube.LabelEnvID:                spec.BuildEnvID,
		kube.LabelEnvSlug:              spec.BuildEnvSlug,
		kube.LabelManagedBy:            "true",
		builderWorkspaceMarkerLabelKey: "true",
		builderSessionIDLabelKey:       spec.SessionID,
	}
}

func validateBuilderWorkspaceStorageSpec(spec BuilderWorkspaceSpec) error {
	if err := validateBuilderWorkspaceName(spec.SessionID); err != nil {
		return err
	}

	if strings.TrimSpace(spec.StorageRequest) == "" {
		return errors.New("builder workspace storage request is required")
	}

	return nil
}

func validateBuilderWorkspacePodSpec(spec BuilderWorkspaceSpec) error {
	if strings.TrimSpace(app.Config.BuilderWorkspaceImage) == "" {
		return errors.New("builder workspace image is required")
	}

	if strings.TrimSpace(app.Config.BuilderWorkspaceRoot) == "" {
		return errors.New("builder workspace root is required")
	}

	return validateBuilderWorkspaceName(spec.SessionID)
}

func validateBuilderWorkspaceName(sessionID string) error {
	if validationErrors := validation.IsDNS1123Subdomain(builderWorkspacePVCName(sessionID)); len(validationErrors) > 0 {
		return fmt.Errorf(
			"builder workspace session ID %q is invalid for resource names: %s",
			sessionID,
			strings.Join(validationErrors, ", "),
		)
	}

	return nil
}

func builderWorkspacePVCName(sessionID string) string {
	return fmt.Sprintf("builder-workspace-%s", sessionID)
}

func builderWorkspacePodName(sessionID string) string {
	return fmt.Sprintf("builder-workspace-%s", sessionID)
}
