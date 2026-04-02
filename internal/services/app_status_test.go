package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetAppStatus_RecalculatesLiveStatusWhenDeployStatusStoredAsRunning(t *testing.T) {
	server := newAppStatusKubeAPIServer(t)
	defer server.Close()

	clusterID := registerAppStatusTestCluster(t, server.URL)

	status := GetAppStatus(context.Background(), &models.AppContext{
		App: entities.App{
			Base:         entities.Base{ID: "app-1"},
			Slug:         "test-app",
			AppType:      app.AppTypeDeployment,
			DeployStatus: app.AppStatusRunning,
		},
		EnvContext: models.EnvContext{
			Env: entities.Env{
				ClusterID:        clusterID,
				ClusterNamespace: "test-ns",
			},
		},
	})

	require.Equal(t, string(app.AppStatusStopped), status)
}

func TestGetAppListRowStatus_RecalculatesLiveStatusWhenDeployStatusStoredAsRunning(t *testing.T) {
	server := newAppStatusKubeAPIServer(t)
	defer server.Close()

	clusterID := registerAppStatusTestCluster(t, server.URL)

	status := GetAppListRowStatus(context.Background(), &models.AppListRow{
		ID:               "app-1",
		Slug:             "test-app",
		AppType:          app.AppTypeDeployment,
		ClusterID:        clusterID,
		ClusterNamespace: "test-ns",
		DeployStatus:     app.AppStatusRunning,
		Replicas:         1,
	})

	require.Equal(t, string(app.AppStatusStopped), status)
}

func TestBuildAppListStatuses_BatchesManagedResourcesByNamespace(t *testing.T) {
	server, counts := newBatchedAppStatusKubeAPIServer(t)
	defer server.Close()

	clusterID := registerAppStatusTestCluster(t, server.URL)

	statuses := BuildAppListStatuses(context.Background(), []models.AppListRow{
		{
			ID:               "app-1",
			Slug:             "app-a",
			AppType:          app.AppTypeDeployment,
			ClusterID:        clusterID,
			ClusterNamespace: "test-ns",
			DeployStatus:     app.AppStatusRunning,
		},
		{
			ID:               "app-2",
			Slug:             "app-b",
			AppType:          app.AppTypeStatefulSet,
			ClusterID:        clusterID,
			ClusterNamespace: "test-ns",
			DeployStatus:     "deployed",
		},
	})

	require.Equal(t, string(app.AppStatusRunning), statuses["app-1"])
	require.Equal(t, string(app.AppStatusDebugging), statuses["app-2"])
	require.Equal(t, 1, counts.pods)
	require.Equal(t, 1, counts.deployments)
	require.Equal(t, 1, counts.statefulSets)
}

func registerAppStatusTestCluster(t *testing.T, serverURL string) string {
	t.Helper()

	clusterID := "cluster-" + t.Name()
	kubeConfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
users:
- name: test
  user: {}
`, serverURL)

	require.NoError(t, kube.GlobalClusterStore.AddClient(clusterID, kubeConfig))
	t.Cleanup(func() {
		kube.GlobalClusterStore.RemoveClient(clusterID)
	})

	return clusterID
}

func newAppStatusKubeAPIServer(t *testing.T) *httptest.Server {
	t.Helper()

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(0),
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          0,
			ReadyReplicas:     0,
			AvailableReplicas: 0,
			UpdatedReplicas:   0,
		},
	}

	podList := &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PodList",
		},
		Items: []corev1.Pod{},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/namespaces/test-ns/pods":
			require.Equal(t, "GET", r.Method)
			require.Equal(t, kube.LabelAppSlug+"=test-app", r.URL.Query().Get("labelSelector"))
			require.NoError(t, json.NewEncoder(w).Encode(podList))
		case "/apis/apps/v1/namespaces/test-ns/deployments/test-app":
			require.Equal(t, "GET", r.Method)
			require.NoError(t, json.NewEncoder(w).Encode(deployment))
		default:
			http.NotFound(w, r)
		}
	}))
}

type batchedAppStatusRequestCount struct {
	pods         int
	deployments  int
	statefulSets int
}

func newBatchedAppStatusKubeAPIServer(t *testing.T) (*httptest.Server, *batchedAppStatusRequestCount) {
	t.Helper()

	counts := &batchedAppStatusRequestCount{}
	expectedSelector := kube.LabelManagedBy + "=true," + kube.LabelAppSlug + " in (app-a,app-b)"

	deploymentList := &appsv1.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DeploymentList",
		},
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-a",
					Namespace: "test-ns",
					Labels: map[string]string{
						kube.LabelManagedBy: "true",
						kube.LabelAppSlug:   "app-a",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          1,
					ReadyReplicas:     1,
					AvailableReplicas: 1,
					UpdatedReplicas:   1,
				},
			},
		},
	}

	statefulSetList := &appsv1.StatefulSetList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSetList",
		},
		Items: []appsv1.StatefulSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-b",
					Namespace: "test-ns",
					Labels: map[string]string{
						kube.LabelManagedBy: "true",
						kube.LabelAppSlug:   "app-b",
					},
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32Ptr(1),
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					UpdatedReplicas: 1,
				},
			},
		},
	}

	podList := &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PodList",
		},
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-b-0",
					Namespace: "test-ns",
					Labels: map[string]string{
						kube.LabelManagedBy: "true",
						kube.LabelAppSlug:   "app-b",
						kube.LabelDebugging: "true",
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/namespaces/test-ns/pods":
			counts.pods++
			require.Equal(t, "GET", r.Method)
			require.Equal(t, expectedSelector, r.URL.Query().Get("labelSelector"))
			require.NoError(t, json.NewEncoder(w).Encode(podList))
		case "/apis/apps/v1/namespaces/test-ns/deployments":
			counts.deployments++
			require.Equal(t, "GET", r.Method)
			require.Equal(t, expectedSelector, r.URL.Query().Get("labelSelector"))
			require.NoError(t, json.NewEncoder(w).Encode(deploymentList))
		case "/apis/apps/v1/namespaces/test-ns/statefulsets":
			counts.statefulSets++
			require.Equal(t, "GET", r.Method)
			require.Equal(t, expectedSelector, r.URL.Query().Get("labelSelector"))
			require.NoError(t, json.NewEncoder(w).Encode(statefulSetList))
		default:
			http.NotFound(w, r)
		}
	}))

	return server, counts
}

func int32Ptr(v int32) *int32 {
	return &v
}
