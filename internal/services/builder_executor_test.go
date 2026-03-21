package services

import (
	"context"
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type fakeBuilderWorkspaceExecutor struct {
	request         builderWorkspaceExecutorRequest
	ensureSnapshot  *builderExecutorSnapshot
	ensureSnapshots map[string]*builderExecutorSnapshot
	ensureErr       error
	ensureErrs      map[string]error
	ensureCalls     []string
	querySnapshot   *builderExecutorSnapshot
	querySnapshots  map[string]*builderExecutorSnapshot
	queryErr        error
	queryErrs       map[string]error
	queryCalls      []string
	cancelErr       error
	cancelErrs      map[string]error
	cancelCalls     []string
	callLog         []string
}

func (f *fakeBuilderWorkspaceExecutor) EnsureWorkspace(ctx context.Context, request builderWorkspaceExecutorRequest) (*builderExecutorSnapshot, error) {
	f.request = request
	handleID := ""
	if request.Handle != nil {
		handleID = request.Handle.ID
	}
	f.ensureCalls = append(f.ensureCalls, handleID)
	f.callLog = append(f.callLog, "ensure:"+handleID)
	if err, ok := f.ensureErrs[handleID]; ok {
		return nil, err
	}
	if snapshot, ok := f.ensureSnapshots[handleID]; ok {
		return snapshot, f.ensureErr
	}
	return f.ensureSnapshot, f.ensureErr
}

func (f *fakeBuilderWorkspaceExecutor) GetWorkspaceSnapshot(ctx context.Context, handle *entities.BuilderExecutorHandle) (*builderExecutorSnapshot, error) {
	handleID := ""
	if handle != nil {
		handleID = handle.ID
	}
	f.queryCalls = append(f.queryCalls, handleID)
	f.callLog = append(f.callLog, "query:"+handleID)
	if err, ok := f.queryErrs[handleID]; ok {
		return nil, err
	}
	if snapshot, ok := f.querySnapshots[handleID]; ok {
		return snapshot, f.queryErr
	}
	return f.querySnapshot, f.queryErr
}

func (f *fakeBuilderWorkspaceExecutor) CancelWorkspace(ctx context.Context, handle *entities.BuilderExecutorHandle) error {
	handleID := ""
	if handle != nil {
		handleID = handle.ID
	}
	f.cancelCalls = append(f.cancelCalls, handleID)
	f.callLog = append(f.callLog, "cancel:"+handleID)
	if err, ok := f.cancelErrs[handleID]; ok {
		return err
	}
	return f.cancelErr
}

func TestBuilderExecutorContract(t *testing.T) {
	fake := &fakeBuilderWorkspaceExecutor{
		ensureSnapshot: &builderExecutorSnapshot{
			HandleID:      "handle-1",
			Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
			Status:        entities.BuilderExecutorHandleStatusActive,
			ClusterID:     "cluster-1",
			Namespace:     "builder-ns",
			WorkloadName:  "builder-workspace-session-1",
			ContainerName: "workspace",
		},
		querySnapshot: &builderExecutorSnapshot{
			HandleID:      "handle-1",
			Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
			Status:        entities.BuilderExecutorHandleStatusFailed,
			ClusterID:     "cluster-1",
			Namespace:     "builder-ns",
			WorkloadName:  "builder-workspace-session-1",
			ContainerName: "workspace",
		},
	}

	var executor builderWorkspaceExecutor = fake
	handle := &entities.BuilderExecutorHandle{
		ID:        "handle-1",
		SessionID: "session-1",
		Kind:      entities.BuilderExecutorHandleKindWorkspacePod,
	}

	snapshot, err := executor.EnsureWorkspace(context.Background(), builderWorkspaceExecutorRequest{
		Handle:         handle,
		SessionID:      "session-1",
		ProjectID:      "project-1",
		ProjectSlug:    "demo",
		BuildEnvID:     "env-1",
		BuildEnvSlug:   "build-env",
		ClusterID:      "cluster-1",
		Namespace:      "builder-ns",
		StorageRequest: builderWorkspaceStorageRequest,
	})
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	assert.Equal(t, []string{"handle-1"}, fake.ensureCalls)
	assert.Equal(t, "session-1", fake.request.SessionID)
	assert.Equal(t, "project-1", fake.request.ProjectID)
	assert.Equal(t, "handle-1", snapshot.HandleID)
	assert.Equal(t, entities.BuilderExecutorHandleKindWorkspacePod, snapshot.Kind)
	assert.Equal(t, entities.BuilderExecutorHandleStatusActive, snapshot.Status)
	assert.Equal(t, "builder-workspace-session-1", snapshot.WorkloadName)
	assert.Equal(t, "workspace", snapshot.ContainerName)
	assert.True(t, snapshot.Usable())
	assert.False(t, snapshot.Terminal())

	queried, err := executor.GetWorkspaceSnapshot(context.Background(), handle)
	require.NoError(t, err)
	require.NotNil(t, queried)
	assert.Equal(t, []string{"handle-1"}, fake.queryCalls)
	assert.Equal(t, entities.BuilderExecutorHandleStatusFailed, queried.Status)
	assert.False(t, queried.Usable())
	assert.True(t, queried.Terminal())

	err = executor.CancelWorkspace(context.Background(), handle)
	require.NoError(t, err)
	assert.Equal(t, []string{"handle-1"}, fake.cancelCalls)
	assert.Equal(t, []string{"ensure:handle-1", "query:handle-1", "cancel:handle-1"}, fake.callLog)
}

func TestBuilderExecutorReconcileSnapshot(t *testing.T) {
	t.Run("reconciles a workspace anchor from the executor snapshot", func(t *testing.T) {
		workspace := &entities.BuilderWorkspace{
			ID:        "workspace-1",
			SessionID: "session-1",
		}

		err := reconcileBuilderWorkspaceFromExecutorSnapshot(workspace, "env-1", "/workspace", &builderExecutorSnapshot{
			HandleID:      "handle-1",
			Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
			Status:        entities.BuilderExecutorHandleStatusActive,
			ClusterID:     "cluster-1",
			Namespace:     "builder-ns",
			WorkloadName:  "builder-workspace-session-1",
			ContainerName: "workspace",
		})
		require.NoError(t, err)

		require.NotNil(t, workspace.ExecutorHandleID)
		assert.Equal(t, "handle-1", *workspace.ExecutorHandleID)
		assert.Equal(t, "env-1", workspace.BuildEnvID)
		assert.Equal(t, "cluster-1", workspace.ClusterID)
		assert.Equal(t, "builder-ns", workspace.Namespace)
		assert.Equal(t, "builder-workspace-session-1", workspace.PodName)
		assert.Equal(t, "workspace", workspace.ContainerName)
		assert.Equal(t, entities.BuilderWorkspaceStatusActive, workspace.Status)
		assert.Equal(t, "/workspace", workspace.WorkspaceRoot)
	})

	t.Run("terminal-state query stays at the interface boundary", func(t *testing.T) {
		executor := &fakeBuilderWorkspaceExecutor{
			querySnapshot: &builderExecutorSnapshot{
				HandleID:      "handle-1",
				Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
				Status:        entities.BuilderExecutorHandleStatusTerminated,
				ClusterID:     "cluster-1",
				Namespace:     "builder-ns",
				WorkloadName:  "builder-workspace-session-1",
				ContainerName: "workspace",
			},
		}

		snapshot, err := executor.GetWorkspaceSnapshot(context.Background(), &entities.BuilderExecutorHandle{
			ID:            "handle-1",
			SessionID:     "session-1",
			Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
			ClusterID:     builderStringPtr("cluster-1"),
			Namespace:     builderStringPtr("builder-ns"),
			WorkloadName:  builderStringPtr("builder-workspace-session-1"),
			ContainerName: builderStringPtr("workspace"),
		})
		require.NoError(t, err)
		require.NotNil(t, snapshot)
		assert.True(t, snapshot.Terminal())
		assert.False(t, snapshot.Usable())
	})

	t.Run("workspace pod carrier maps onto the current pod path", func(t *testing.T) {
		setBuilderWorkspaceServiceConfigForTest(t)

		client := kubefake.NewSimpleClientset()
		client.Fake.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			createAction := action.(k8stesting.CreateAction)
			pod := createAction.GetObject().(*corev1.Pod)
			pod.Status.Conditions = []corev1.PodCondition{{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}}
			return false, nil, nil
		})

		originalGetBuilderWorkspaceClusterClient := getBuilderWorkspaceClusterClient
		t.Cleanup(func() {
			getBuilderWorkspaceClusterClient = originalGetBuilderWorkspaceClusterClient
		})
		getBuilderWorkspaceClusterClient = func(clusterID string) (kubernetes.Interface, error) {
			require.Equal(t, "cluster-1", clusterID)
			return client, nil
		}

		executor := builderWorkspacePodExecutorV1{}
		snapshot, err := executor.EnsureWorkspace(context.Background(), builderWorkspaceExecutorRequest{
			Handle: &entities.BuilderExecutorHandle{
				ID:        "handle-1",
				SessionID: "session-1",
				Kind:      entities.BuilderExecutorHandleKindWorkspacePod,
			},
			SessionID:      "session-1",
			ProjectID:      "project-1",
			ProjectSlug:    "demo",
			BuildEnvID:     "env-1",
			BuildEnvSlug:   "build-env",
			ClusterID:      "cluster-1",
			Namespace:      "builder-ns",
			StorageRequest: builderWorkspaceStorageRequest,
		})
		require.NoError(t, err)
		require.NotNil(t, snapshot)

		assert.Equal(t, "handle-1", snapshot.HandleID)
		assert.Equal(t, entities.BuilderExecutorHandleKindWorkspacePod, snapshot.Kind)
		assert.Equal(t, entities.BuilderExecutorHandleStatusActive, snapshot.Status)
		assert.Equal(t, "cluster-1", snapshot.ClusterID)
		assert.Equal(t, "builder-ns", snapshot.Namespace)
		assert.Equal(t, "builder-workspace-session-1", snapshot.WorkloadName)
		assert.Equal(t, "workspace", snapshot.ContainerName)
		assert.True(t, snapshot.Usable())
		assert.False(t, snapshot.Terminal())

		queried, err := executor.GetWorkspaceSnapshot(context.Background(), &entities.BuilderExecutorHandle{
			ID:            snapshot.HandleID,
			SessionID:     "session-1",
			Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
			ClusterID:     builderStringPtr(snapshot.ClusterID),
			Namespace:     builderStringPtr(snapshot.Namespace),
			WorkloadName:  builderStringPtr(snapshot.WorkloadName),
			ContainerName: builderStringPtr(snapshot.ContainerName),
		})
		require.NoError(t, err)
		require.NotNil(t, queried)
		assert.Equal(t, entities.BuilderExecutorHandleStatusActive, queried.Status)
		assert.True(t, queried.Usable())
		assert.False(t, queried.Terminal())

		_, err = client.CoreV1().PersistentVolumeClaims("builder-ns").Get(context.Background(), snapshot.WorkloadName, metav1.GetOptions{})
		require.NoError(t, err)
		_, err = client.CoreV1().Pods("builder-ns").Get(context.Background(), snapshot.WorkloadName, metav1.GetOptions{})
		require.NoError(t, err)
	})
}
