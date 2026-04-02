package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupAppListResponseServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.Cluster{},
		&entities.Env{},
		&entities.App{},
		&entities.AppFavorite{},
		&entities.AppGroup{},
		&entities.AppGroupMember{},
	))

	db.DB = testDB
}

func seedAppListResponseFixture(t *testing.T) (clusterID, envID, groupID string) {
	t.Helper()

	clusterID = "cluster-batch"
	envID = "env-batch"
	groupID = "group-batch"

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: clusterID},
		Slug:       clusterID,
		Name:       "Cluster Batch",
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: envID},
		Slug:             envID,
		Name:             "Env Batch",
		ProjectID:        "project-1",
		ClusterID:        clusterID,
		ClusterNamespace: "batch-ns",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:         entities.Base{ID: "app-a"},
		Slug:         "app-a",
		Name:         "App A",
		EnvID:        envID,
		AppType:      app.AppTypeDeployment,
		DeployStatus: app.AppStatusRunning,
		Replicas:     1,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:         entities.Base{ID: "app-b"},
		Slug:         "app-b",
		Name:         "App B",
		EnvID:        envID,
		AppType:      app.AppTypeStatefulSet,
		DeployStatus: "deployed",
		Replicas:     1,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppFavorite{
		ID:     "fav-a",
		UserID: "user-1",
		EnvID:  envID,
		AppID:  "app-a",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppFavorite{
		ID:     "fav-b",
		UserID: "user-1",
		EnvID:  envID,
		AppID:  "app-b",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGroup{
		ID:    groupID,
		EnvID: envID,
		Name:  "Group Batch",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGroupMember{
		ID:      "group-member-a",
		GroupID: groupID,
		AppID:   "app-a",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGroupMember{
		ID:      "group-member-b",
		GroupID: groupID,
		AppID:   "app-b",
	}).Error)

	return clusterID, envID, groupID
}

func TestListFavoriteAppsUsesBatchedStatuses(t *testing.T) {
	setupAppListResponseServiceTestDB(t)
	server, counts := newBatchedFavoriteStatusServer(t)
	defer server.Close()

	clusterID, envID, _ := seedAppListResponseFixture(t)
	registerAppListResponseCluster(t, clusterID, server.URL)

	items, err := ListFavoriteApps(context.Background(), "user-1", envID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	statusByID := make(map[string]string, len(items))
	for i := range items {
		statusByID[items[i].ID] = items[i].Status
	}
	assert.Equal(t, string(app.AppStatusRunning), statusByID["app-a"])
	assert.Equal(t, string(app.AppStatusDebugging), statusByID["app-b"])
	assert.Equal(t, 1, counts.pods)
	assert.Equal(t, 1, counts.deployments)
	assert.Equal(t, 1, counts.statefulSets)
}

func TestListSpecificGroupedAppsUsesBatchedStatuses(t *testing.T) {
	setupAppListResponseServiceTestDB(t)
	server, counts := newBatchedFavoriteStatusServer(t)
	defer server.Close()

	clusterID, _, groupID := seedAppListResponseFixture(t)
	registerAppListResponseCluster(t, clusterID, server.URL)

	total, items, err := ListSpecificGroupedApps(context.Background(), groupID, 1, 10, "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	statusByID := make(map[string]string, len(items))
	for i := range items {
		statusByID[items[i].ID] = items[i].Status
	}
	assert.Equal(t, string(app.AppStatusRunning), statusByID["app-a"])
	assert.Equal(t, string(app.AppStatusDebugging), statusByID["app-b"])
	assert.Equal(t, 1, counts.pods)
	assert.Equal(t, 1, counts.deployments)
	assert.Equal(t, 1, counts.statefulSets)
}

func registerAppListResponseCluster(t *testing.T, clusterID, serverURL string) {
	t.Helper()

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
}

type batchedFavoriteStatusRequestCount struct {
	pods         int
	deployments  int
	statefulSets int
}

func newBatchedFavoriteStatusServer(t *testing.T) (*httptest.Server, *batchedFavoriteStatusRequestCount) {
	t.Helper()

	counts := &batchedFavoriteStatusRequestCount{}
	expectedSelector := kube.LabelManagedBy + "=true," + kube.LabelAppSlug + " in (app-a,app-b)"

	deploymentList := &appsv1.DeploymentList{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DeploymentList"},
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-a",
					Namespace: "batch-ns",
					Labels: map[string]string{
						kube.LabelManagedBy: "true",
						kube.LabelAppSlug:   "app-a",
					},
				},
				Spec: appsv1.DeploymentSpec{Replicas: int32Ptr(1)},
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
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSetList"},
		Items: []appsv1.StatefulSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-b",
					Namespace: "batch-ns",
					Labels: map[string]string{
						kube.LabelManagedBy: "true",
						kube.LabelAppSlug:   "app-b",
					},
				},
				Spec: appsv1.StatefulSetSpec{Replicas: int32Ptr(1)},
				Status: appsv1.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					UpdatedReplicas: 1,
				},
			},
		},
	}

	podList := &corev1.PodList{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "PodList"},
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-b-0",
					Namespace: "batch-ns",
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
		case "/api/v1/namespaces/batch-ns/pods":
			counts.pods++
			require.Equal(t, expectedSelector, r.URL.Query().Get("labelSelector"))
			require.NoError(t, json.NewEncoder(w).Encode(podList))
		case "/apis/apps/v1/namespaces/batch-ns/deployments":
			counts.deployments++
			require.Equal(t, expectedSelector, r.URL.Query().Get("labelSelector"))
			require.NoError(t, json.NewEncoder(w).Encode(deploymentList))
		case "/apis/apps/v1/namespaces/batch-ns/statefulsets":
			counts.statefulSets++
			require.Equal(t, expectedSelector, r.URL.Query().Get("labelSelector"))
			require.NoError(t, json.NewEncoder(w).Encode(statefulSetList))
		default:
			http.NotFound(w, r)
		}
	}))

	return server, counts
}
