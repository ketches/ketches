package services

import (
	"context"
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestValidateAppPodAccessRequiresApplicationIdentity(t *testing.T) {
	basePod := func(labels map[string]string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: "app-ns", Labels: labels},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "init"}},
				Containers:     []corev1.Container{{Name: "app-main"}, {Name: "sidecar"}},
			},
		}
	}

	appContext := &models.AppContext{
		App:        entities.App{Base: entities.Base{ID: "app-1"}},
		EnvContext: models.EnvContext{Env: entities.Env{ClusterNamespace: "app-ns"}},
	}

	tests := []struct {
		name      string
		labels    map[string]string
		container string
		wantError bool
	}{
		{
			name: "owned application pod",
			labels: map[string]string{
				kube.LabelManagedBy: "true",
				kube.LabelAppID:     "app-1",
			},
			container: "sidecar",
		},
		{
			name: "owned application init container",
			labels: map[string]string{
				kube.LabelManagedBy: "true",
				kube.LabelAppID:     "app-1",
			},
			container: "init",
		},
		{
			name: "unmanaged pod",
			labels: map[string]string{
				kube.LabelAppID: "app-1",
			},
			container: "app-main",
			wantError: true,
		},
		{
			name: "other application pod",
			labels: map[string]string{
				kube.LabelManagedBy: "true",
				kube.LabelAppID:     "app-2",
			},
			container: "app-main",
			wantError: true,
		},
		{
			name: "build pod",
			labels: map[string]string{
				kube.LabelManagedBy: "true",
				kube.LabelAppID:     "app-1",
				kube.LabelBuildKey:  "true",
			},
			container: "app-main",
			wantError: true,
		},
		{
			name: "unknown container",
			labels: map[string]string{
				kube.LabelManagedBy: "true",
				kube.LabelAppID:     "app-1",
			},
			container: "not-in-pod",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(basePod(tt.labels))
			pod, err := validateAppPodContainer(context.Background(), client, appContext, "app-pod", tt.container)
			if tt.wantError {
				require.Error(t, err)
				require.ErrorIs(t, err, errPodAccessDenied)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "app-pod", pod.Name)
		})
	}
}

func TestValidateAppPodAccessUsesScopedPolicyForBuilderWorkspace(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workspace-pod",
			Namespace: "builder-ns",
			Labels: map[string]string{
				kube.LabelManagedBy:        "true",
				kube.LabelBuilderWorkspace: "true",
				kube.LabelBuilderSessionID: "session-1",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "workspace"}}},
	})
	appContext := &models.AppContext{
		EnvContext: models.EnvContext{Env: entities.Env{ClusterNamespace: "builder-ns"}},
		PodAccessPolicy: &models.PodAccessPolicy{RequiredLabels: map[string]string{
			kube.LabelBuilderWorkspace: "true",
			kube.LabelBuilderSessionID: "session-1",
		}},
	}

	_, err := validateAppPodContainer(context.Background(), client, appContext, "workspace-pod", "workspace")
	require.NoError(t, err)

	appContext.PodAccessPolicy.RequiredLabels[kube.LabelBuilderSessionID] = "session-2"
	_, err = validateAppPodAccess(context.Background(), client, appContext, "workspace-pod")
	require.ErrorIs(t, err, errPodAccessDenied)
}
