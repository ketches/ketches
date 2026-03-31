package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func setBuilderWorkspaceExecutorFactoryForTest(t *testing.T, factory func(kind entities.BuilderExecutorHandleKind) (builderWorkspaceExecutor, error)) {
	t.Helper()

	originalFactory := getBuilderWorkspaceExecutor
	getBuilderWorkspaceExecutor = factory
	t.Cleanup(func() {
		getBuilderWorkspaceExecutor = originalFactory
	})
}

func setupBuilderWorkspaceServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	db.DB = testDB
	require.NoError(t, db.Migrate())
}

func setBuilderWorkspaceServiceConfigForTest(t *testing.T) {
	t.Helper()

	originalConfig := app.Config
	app.Config = app.AppConfig{
		BuilderWorkspaceImage: "ghcr.io/ketches/builder-workspace:latest",
		BuilderWorkspaceRoot:  "/workspace",
	}
	t.Cleanup(func() {
		app.Config = originalConfig
	})
}

func seedBuilderWorkspaceServiceFixture(t *testing.T) (*entities.BuilderSession, *entities.BuilderWorkspace, *entities.BuilderRun) {
	t.Helper()

	now := time.Now().UTC()

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo",
		Name: "Demo Project",
	}).Error)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-1",
		Name:       "Cluster 1",
		KubeConfig: "apiVersion: v1",
	}).Error)

	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "build-env",
		Name:             "Build Env",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "builder-ns",
		IsBuildEnv:       true,
	}).Error)

	session := &entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-1",
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Status:         entities.BuilderSessionStatusProvisioning,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(session).Error)

	workspace := &entities.BuilderWorkspace{
		ID:            "workspace-1",
		CreatedAt:     now.Add(-4 * time.Minute),
		UpdatedAt:     now.Add(-4 * time.Minute),
		SessionID:     session.ID,
		BuildEnvID:    "env-1",
		ClusterID:     "cluster-1",
		Namespace:     "builder-ns",
		PodName:       "builder-workspace-session-1",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: "/workspace",
	}
	if err := db.DB.Create(workspace).Error; err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
		require.NoError(t, err)
	}

	runWorkspaceID := workspace.ID
	run := &entities.BuilderRun{
		ID:                 "run-1",
		CreatedAt:          now.Add(-3 * time.Minute),
		UpdatedAt:          now.Add(-3 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   "message-1",
		WorkspaceID:        &runWorkspaceID,
		Status:             entities.BuilderRunStatusExecuting,
		RequestedBy:        "user-1",
		InstructionSummary: "Create the initial project structure.",
	}
	require.NoError(t, db.DB.Create(run).Error)

	return session, workspace, run
}

func markBuilderWorkspacePodReadyOnCreate(client *kubefake.Clientset) {
	client.Fake.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		pod := createAction.GetObject().(*corev1.Pod)
		pod.Status.Conditions = []corev1.PodCondition{{
			Type:   corev1.PodReady,
			Status: corev1.ConditionTrue,
		}}
		return false, nil, nil
	})
}

func addReadyBuilderWorkspacePod(t *testing.T, client kubernetes.Interface, namespace, podName, containerName string) {
	t.Helper()

	_, err := client.CoreV1().Pods(namespace).Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: containerName}},
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
}

func TestProvisionBuilderWorkspaceCreatesPodAndPVCRecords(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	session, _, _ := seedBuilderWorkspaceServiceFixture(t)

	require.NoError(t, db.DB.Where("session_id = ?", session.ID).Delete(&entities.BuilderWorkspace{}).Error)

	client := kubefake.NewSimpleClientset()
	markBuilderWorkspacePodReadyOnCreate(client)
	originalGetBuilderWorkspaceClusterClient := getBuilderWorkspaceClusterClient
	t.Cleanup(func() {
		getBuilderWorkspaceClusterClient = originalGetBuilderWorkspaceClusterClient
	})
	getBuilderWorkspaceClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		require.Equal(t, "cluster-1", clusterID)
		return client, nil
	}

	workspace, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, workspace)

	assert.Equal(t, session.ID, workspace.SessionID)
	assert.Equal(t, "env-1", workspace.BuildEnvID)
	assert.Equal(t, "cluster-1", workspace.ClusterID)
	assert.Equal(t, "builder-ns", workspace.Namespace)
	assert.Equal(t, app.Config.BuilderWorkspaceRoot, workspace.WorkspaceRoot)
	assert.Equal(t, entities.BuilderWorkspaceStatusActive, workspace.Status)

	stored := &entities.BuilderWorkspace{}
	require.NoError(t, db.DB.First(stored, "id = ?", workspace.ID).Error)
	assert.Equal(t, workspace.PodName, stored.PodName)
	assert.Equal(t, entities.BuilderWorkspaceStatusActive, stored.Status)

	updatedSession := &entities.BuilderSession{}
	require.NoError(t, db.DB.First(updatedSession, "id = ?", session.ID).Error)
	assert.Equal(t, entities.BuilderSessionStatusReady, updatedSession.Status)

	_, err = client.CoreV1().PersistentVolumeClaims("builder-ns").Get(context.Background(), workspace.PodName, metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().Pods("builder-ns").Get(context.Background(), workspace.PodName, metav1.GetOptions{})
	require.NoError(t, err)
}

func TestProvisionBuilderWorkspaceCreatesExecutorHandle(t *testing.T) {
	t.Run("creates and reuses one active session-scoped workspace handle", func(t *testing.T) {
		setupBuilderWorkspaceServiceTestDB(t)
		setBuilderWorkspaceServiceConfigForTest(t)
		session, _, _ := seedBuilderWorkspaceServiceFixture(t)

		require.NoError(t, db.DB.Where("session_id = ?", session.ID).Delete(&entities.BuilderWorkspace{}).Error)

		client := kubefake.NewSimpleClientset()
		markBuilderWorkspacePodReadyOnCreate(client)
		originalGetBuilderWorkspaceClusterClient := getBuilderWorkspaceClusterClient
		t.Cleanup(func() {
			getBuilderWorkspaceClusterClient = originalGetBuilderWorkspaceClusterClient
		})
		getBuilderWorkspaceClusterClient = func(clusterID string) (kubernetes.Interface, error) {
			require.Equal(t, "cluster-1", clusterID)
			return client, nil
		}

		workspace, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
		require.NoError(t, err)
		require.NotNil(t, workspace)
		require.NotNil(t, workspace.ExecutorHandleID)

		storedHandle := &entities.BuilderExecutorHandle{}
		require.NoError(t, db.DB.First(storedHandle, "id = ?", *workspace.ExecutorHandleID).Error)
		assert.Equal(t, session.ID, storedHandle.SessionID)
		assert.Nil(t, storedHandle.RunID)
		assert.Equal(t, entities.BuilderExecutorHandleKindWorkspacePod, storedHandle.Kind)
		assert.Equal(t, entities.BuilderExecutorHandleStatusActive, storedHandle.Status)
		require.NotNil(t, storedHandle.ClusterID)
		assert.Equal(t, workspace.ClusterID, *storedHandle.ClusterID)
		require.NotNil(t, storedHandle.Namespace)
		assert.Equal(t, workspace.Namespace, *storedHandle.Namespace)
		require.NotNil(t, storedHandle.WorkloadName)
		assert.Equal(t, workspace.PodName, *storedHandle.WorkloadName)
		require.NotNil(t, storedHandle.ContainerName)
		assert.Equal(t, workspace.ContainerName, *storedHandle.ContainerName)

		require.NoError(t, db.DB.Where("session_id = ?", session.ID).Delete(&entities.BuilderWorkspace{}).Error)
		restoredWorkspace, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
		require.NoError(t, err)
		require.NotNil(t, restoredWorkspace)
		require.NotNil(t, restoredWorkspace.ExecutorHandleID)
		assert.Equal(t, *workspace.ExecutorHandleID, *restoredWorkspace.ExecutorHandleID)

		var handleCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderExecutorHandle{}).Where("session_id = ? AND kind = ? AND status = ?", session.ID, entities.BuilderExecutorHandleKindWorkspacePod, entities.BuilderExecutorHandleStatusActive).Count(&handleCount).Error)
		assert.Equal(t, int64(1), handleCount)
	})

	t.Run("reconciliation leaves only one active session-scoped workspace handle", func(t *testing.T) {
		setupBuilderWorkspaceServiceTestDB(t)
		setBuilderWorkspaceServiceConfigForTest(t)
		session, workspace, _ := seedBuilderWorkspaceServiceFixture(t)

		olderHandle := &entities.BuilderExecutorHandle{
			ID:            "handle-older",
			CreatedAt:     workspace.CreatedAt.Add(-2 * time.Minute),
			UpdatedAt:     workspace.CreatedAt.Add(-2 * time.Minute),
			SessionID:     session.ID,
			Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
			Status:        entities.BuilderExecutorHandleStatusActive,
			ClusterID:     builderStringPtr("cluster-1"),
			Namespace:     builderStringPtr("builder-ns"),
			WorkloadName:  builderStringPtr("builder-workspace-session-1-older"),
			ContainerName: builderStringPtr("workspace"),
		}
		canonicalHandle := &entities.BuilderExecutorHandle{
			ID:            "handle-canonical",
			CreatedAt:     workspace.CreatedAt.Add(-time.Minute),
			UpdatedAt:     workspace.CreatedAt.Add(-time.Minute),
			SessionID:     session.ID,
			Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
			Status:        entities.BuilderExecutorHandleStatusActive,
			ClusterID:     builderStringPtr("cluster-1"),
			Namespace:     builderStringPtr("builder-ns"),
			WorkloadName:  builderStringPtr("builder-workspace-session-1"),
			ContainerName: builderStringPtr("workspace"),
		}
		require.NoError(t, db.DB.Create(olderHandle).Error)
		require.NoError(t, db.DB.Create(canonicalHandle).Error)
		require.NoError(t, db.DB.Model(&entities.BuilderWorkspace{}).Where("id = ?", workspace.ID).Update("executor_handle_id", canonicalHandle.ID).Error)

		fake := &fakeBuilderWorkspaceExecutor{
			querySnapshots: map[string]*builderExecutorSnapshot{
				olderHandle.ID: {
					HandleID:      olderHandle.ID,
					Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
					Status:        entities.BuilderExecutorHandleStatusActive,
					ClusterID:     "cluster-1",
					Namespace:     "builder-ns",
					WorkloadName:  "builder-workspace-session-1-older",
					ContainerName: "workspace",
				},
				canonicalHandle.ID: {
					HandleID:      canonicalHandle.ID,
					Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
					Status:        entities.BuilderExecutorHandleStatusActive,
					ClusterID:     "cluster-1",
					Namespace:     "builder-ns",
					WorkloadName:  "builder-workspace-session-1",
					ContainerName: "workspace",
				},
			},
		}
		setBuilderWorkspaceExecutorFactoryForTest(t, func(kind entities.BuilderExecutorHandleKind) (builderWorkspaceExecutor, error) {
			require.Equal(t, entities.BuilderExecutorHandleKindWorkspacePod, kind)
			return fake, nil
		})

		result, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.ExecutorHandleID)
		assert.Equal(t, canonicalHandle.ID, *result.ExecutorHandleID)
		assert.Equal(t, canonicalHandle.ID, fake.queryCalls[0])
		assert.Equal(t, []string{olderHandle.ID}, fake.cancelCalls)

		var activeHandles []entities.BuilderExecutorHandle
		require.NoError(t, db.DB.Where("session_id = ? AND kind = ? AND status = ?", session.ID, entities.BuilderExecutorHandleKindWorkspacePod, entities.BuilderExecutorHandleStatusActive).Order("id ASC").Find(&activeHandles).Error)
		require.Len(t, activeHandles, 1)
		assert.Equal(t, canonicalHandle.ID, activeHandles[0].ID)

		terminatedHandle := &entities.BuilderExecutorHandle{}
		require.NoError(t, db.DB.First(terminatedHandle, "id = ?", olderHandle.ID).Error)
		assert.Equal(t, entities.BuilderExecutorHandleStatusTerminated, terminatedHandle.Status)
		require.NotNil(t, terminatedHandle.TerminatedAt)
	})

	t.Run("waits for pod readiness before the session becomes ready", func(t *testing.T) {
		setupBuilderWorkspaceServiceTestDB(t)
		setBuilderWorkspaceServiceConfigForTest(t)
		session, _, _ := seedBuilderWorkspaceServiceFixture(t)

		require.NoError(t, db.DB.Where("session_id = ?", session.ID).Delete(&entities.BuilderWorkspace{}).Error)

		client := kubefake.NewSimpleClientset()
		originalGetBuilderWorkspaceClusterClient := getBuilderWorkspaceClusterClient
		t.Cleanup(func() {
			getBuilderWorkspaceClusterClient = originalGetBuilderWorkspaceClusterClient
		})
		getBuilderWorkspaceClusterClient = func(clusterID string) (kubernetes.Interface, error) {
			require.Equal(t, "cluster-1", clusterID)
			return client, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		workspace, err := ProvisionBuilderWorkspace(ctx, session.ID)
		require.Error(t, err)
		assert.Nil(t, workspace)

		var workspaceCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderWorkspace{}).Where("session_id = ?", session.ID).Count(&workspaceCount).Error)
		assert.Equal(t, int64(0), workspaceCount)

		updatedSession := &entities.BuilderSession{}
		require.NoError(t, db.DB.First(updatedSession, "id = ?", session.ID).Error)
		assert.Equal(t, entities.BuilderSessionStatusProvisioning, updatedSession.Status)
	})
}

func TestProvisionBuilderWorkspaceDoesNotTrustStaleWorkspaceRows(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	session, workspace, _ := seedBuilderWorkspaceServiceFixture(t)

	handle := &entities.BuilderExecutorHandle{
		ID:            "handle-1",
		CreatedAt:     workspace.CreatedAt.Add(-time.Minute),
		UpdatedAt:     workspace.CreatedAt.Add(-time.Minute),
		SessionID:     session.ID,
		Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
		Status:        entities.BuilderExecutorHandleStatusActive,
		ClusterID:     builderStringPtr("cluster-1"),
		Namespace:     builderStringPtr("builder-ns"),
		WorkloadName:  builderStringPtr("builder-workspace-session-1"),
		ContainerName: builderStringPtr("workspace"),
	}
	require.NoError(t, db.DB.Create(handle).Error)
	require.NoError(t, db.DB.Model(&entities.BuilderWorkspace{}).Where("id = ?", workspace.ID).Updates(map[string]any{
		"executor_handle_id": handle.ID,
		"status":             entities.BuilderWorkspaceStatusActive,
	}).Error)

	fake := &fakeBuilderWorkspaceExecutor{
		querySnapshots: map[string]*builderExecutorSnapshot{
			handle.ID: {
				HandleID:      handle.ID,
				Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
				Status:        entities.BuilderExecutorHandleStatusProvisioning,
				ClusterID:     "cluster-1",
				Namespace:     "builder-ns",
				WorkloadName:  "builder-workspace-session-1",
				ContainerName: "workspace",
			},
		},
		ensureSnapshots: map[string]*builderExecutorSnapshot{
			handle.ID: {
				HandleID:      handle.ID,
				Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
				Status:        entities.BuilderExecutorHandleStatusActive,
				ClusterID:     "cluster-1",
				Namespace:     "builder-ns",
				WorkloadName:  "builder-workspace-session-1",
				ContainerName: "workspace",
			},
		},
	}
	setBuilderWorkspaceExecutorFactoryForTest(t, func(kind entities.BuilderExecutorHandleKind) (builderWorkspaceExecutor, error) {
		require.Equal(t, entities.BuilderExecutorHandleKindWorkspacePod, kind)
		return fake, nil
	})

	result, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.ExecutorHandleID)
	assert.Equal(t, handle.ID, *result.ExecutorHandleID)
	assert.Equal(t, []string{handle.ID}, fake.queryCalls)
	assert.Equal(t, []string{handle.ID}, fake.ensureCalls)
	assert.Equal(t, []string{"query:handle-1", "ensure:handle-1"}, fake.callLog)

	storedWorkspace := &entities.BuilderWorkspace{}
	require.NoError(t, db.DB.First(storedWorkspace, "id = ?", workspace.ID).Error)
	assert.Equal(t, entities.BuilderWorkspaceStatusActive, storedWorkspace.Status)
	require.NotNil(t, storedWorkspace.ExecutorHandleID)
	assert.Equal(t, handle.ID, *storedWorkspace.ExecutorHandleID)

	updatedSession := &entities.BuilderSession{}
	require.NoError(t, db.DB.First(updatedSession, "id = ?", session.ID).Error)
	assert.Equal(t, entities.BuilderSessionStatusReady, updatedSession.Status)
}

func TestProvisionBuilderWorkspaceRecoversFromIncompleteProvisioningHandle(t *testing.T) {
	t.Run("persisted provisioning handle without runtime refs does not wedge retries", func(t *testing.T) {
		setupBuilderWorkspaceServiceTestDB(t)
		setBuilderWorkspaceServiceConfigForTest(t)
		session, _, _ := seedBuilderWorkspaceServiceFixture(t)

		require.NoError(t, db.DB.Where("session_id = ?", session.ID).Delete(&entities.BuilderWorkspace{}).Error)

		incompleteHandle := &entities.BuilderExecutorHandle{
			ID:        "handle-incomplete",
			CreatedAt: time.Now().UTC().Add(-time.Minute),
			UpdatedAt: time.Now().UTC().Add(-time.Minute),
			SessionID: session.ID,
			Kind:      entities.BuilderExecutorHandleKindWorkspacePod,
			Status:    entities.BuilderExecutorHandleStatusProvisioning,
			ClusterID: builderStringPtr("cluster-1"),
			Namespace: builderStringPtr("builder-ns"),
		}
		require.NoError(t, db.DB.Create(incompleteHandle).Error)

		client := kubefake.NewSimpleClientset()
		markBuilderWorkspacePodReadyOnCreate(client)
		originalGetBuilderWorkspaceClusterClient := getBuilderWorkspaceClusterClient
		t.Cleanup(func() {
			getBuilderWorkspaceClusterClient = originalGetBuilderWorkspaceClusterClient
		})
		getBuilderWorkspaceClusterClient = func(clusterID string) (kubernetes.Interface, error) {
			require.Equal(t, "cluster-1", clusterID)
			return client, nil
		}

		workspace, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
		require.NoError(t, err)
		require.NotNil(t, workspace)
		require.NotNil(t, workspace.ExecutorHandleID)
		assert.Equal(t, incompleteHandle.ID, *workspace.ExecutorHandleID)

		storedHandle := &entities.BuilderExecutorHandle{}
		require.NoError(t, db.DB.First(storedHandle, "id = ?", incompleteHandle.ID).Error)
		assert.Equal(t, entities.BuilderExecutorHandleStatusActive, storedHandle.Status)
		require.NotNil(t, storedHandle.WorkloadName)
		assert.Equal(t, workspace.PodName, *storedHandle.WorkloadName)
		require.NotNil(t, storedHandle.ContainerName)
		assert.Equal(t, workspace.ContainerName, *storedHandle.ContainerName)

		var activeHandleCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderExecutorHandle{}).Where("session_id = ? AND kind = ? AND status = ?", session.ID, entities.BuilderExecutorHandleKindWorkspacePod, entities.BuilderExecutorHandleStatusActive).Count(&activeHandleCount).Error)
		assert.Equal(t, int64(1), activeHandleCount)
	})

	t.Run("malformed stale handle does not block a healthy handle", func(t *testing.T) {
		setupBuilderWorkspaceServiceTestDB(t)
		setBuilderWorkspaceServiceConfigForTest(t)
		session, _, _ := seedBuilderWorkspaceServiceFixture(t)

		require.NoError(t, db.DB.Where("session_id = ?", session.ID).Delete(&entities.BuilderWorkspace{}).Error)

		healthyHandle := &entities.BuilderExecutorHandle{
			ID:            "handle-healthy",
			CreatedAt:     time.Now().UTC().Add(-2 * time.Minute),
			UpdatedAt:     time.Now().UTC().Add(-2 * time.Minute),
			SessionID:     session.ID,
			Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
			Status:        entities.BuilderExecutorHandleStatusActive,
			ClusterID:     builderStringPtr("cluster-1"),
			Namespace:     builderStringPtr("builder-ns"),
			WorkloadName:  builderStringPtr("builder-workspace-session-1-healthy"),
			ContainerName: builderStringPtr("workspace"),
		}
		malformedHandle := &entities.BuilderExecutorHandle{
			ID:        "handle-malformed",
			CreatedAt: time.Now().UTC().Add(-time.Minute),
			UpdatedAt: time.Now().UTC().Add(-time.Minute),
			SessionID: session.ID,
			Kind:      entities.BuilderExecutorHandleKindWorkspacePod,
			Status:    entities.BuilderExecutorHandleStatusProvisioning,
			ClusterID: builderStringPtr("cluster-1"),
			Namespace: builderStringPtr("builder-ns"),
		}
		require.NoError(t, db.DB.Create(healthyHandle).Error)
		require.NoError(t, db.DB.Create(malformedHandle).Error)

		client := kubefake.NewSimpleClientset()
		addReadyBuilderWorkspacePod(t, client, "builder-ns", "builder-workspace-session-1-healthy", "workspace")
		client.Fake.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			deleteAction := action.(k8stesting.DeleteAction)
			if deleteAction.GetName() == "builder-workspace-session-1-healthy" {
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, deleteAction.GetName(), errors.New("healthy handle should not be cancelled"))
			}
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

		workspace, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
		require.NoError(t, err)
		require.NotNil(t, workspace)
		require.NotNil(t, workspace.ExecutorHandleID)
		assert.Equal(t, healthyHandle.ID, *workspace.ExecutorHandleID)

		storedHealthy := &entities.BuilderExecutorHandle{}
		require.NoError(t, db.DB.First(storedHealthy, "id = ?", healthyHandle.ID).Error)
		assert.Equal(t, entities.BuilderExecutorHandleStatusActive, storedHealthy.Status)

		storedMalformed := &entities.BuilderExecutorHandle{}
		require.NoError(t, db.DB.First(storedMalformed, "id = ?", malformedHandle.ID).Error)
		assert.Equal(t, entities.BuilderExecutorHandleStatusTerminated, storedMalformed.Status)
		require.NotNil(t, storedMalformed.TerminatedAt)

		var activeHandleCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderExecutorHandle{}).Where("session_id = ? AND kind = ? AND status = ?", session.ID, entities.BuilderExecutorHandleKindWorkspacePod, entities.BuilderExecutorHandleStatusActive).Count(&activeHandleCount).Error)
		assert.Equal(t, int64(1), activeHandleCount)
	})
}

func TestProvisionBuilderWorkspaceDoesNotCancelEquivalentDuplicateHandle(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	session, workspace, _ := seedBuilderWorkspaceServiceFixture(t)

	healthyHandle := &entities.BuilderExecutorHandle{
		ID:            "handle-healthy",
		CreatedAt:     workspace.CreatedAt.Add(-2 * time.Minute),
		UpdatedAt:     workspace.CreatedAt.Add(-2 * time.Minute),
		SessionID:     session.ID,
		Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
		Status:        entities.BuilderExecutorHandleStatusActive,
		ClusterID:     builderStringPtr("cluster-1"),
		Namespace:     builderStringPtr("builder-ns"),
		WorkloadName:  builderStringPtr("builder-workspace-session-1"),
		ContainerName: builderStringPtr("workspace"),
	}
	duplicateHandle := &entities.BuilderExecutorHandle{
		ID:            "handle-duplicate",
		CreatedAt:     workspace.CreatedAt.Add(-time.Minute),
		UpdatedAt:     workspace.CreatedAt.Add(-time.Minute),
		SessionID:     session.ID,
		Kind:          entities.BuilderExecutorHandleKindWorkspacePod,
		Status:        entities.BuilderExecutorHandleStatusActive,
		ClusterID:     builderStringPtr("cluster-1"),
		Namespace:     builderStringPtr("builder-ns"),
		WorkloadName:  builderStringPtr("builder-workspace-session-1"),
		ContainerName: builderStringPtr("stale-container-name"),
	}
	require.NoError(t, db.DB.Create(healthyHandle).Error)
	require.NoError(t, db.DB.Create(duplicateHandle).Error)
	require.NoError(t, db.DB.Model(&entities.BuilderWorkspace{}).Where("id = ?", workspace.ID).Update("executor_handle_id", healthyHandle.ID).Error)

	client := kubefake.NewSimpleClientset()
	addReadyBuilderWorkspacePod(t, client, "builder-ns", "builder-workspace-session-1", "workspace")
	client.Fake.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteAction := action.(k8stesting.DeleteAction)
		if deleteAction.GetName() == "builder-workspace-session-1" {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, deleteAction.GetName(), errors.New("equivalent duplicate handle with different container name should not cancel canonical pod"))
		}
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

	result, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.ExecutorHandleID)
	assert.Equal(t, healthyHandle.ID, *result.ExecutorHandleID)

	storedHealthy := &entities.BuilderExecutorHandle{}
	require.NoError(t, db.DB.First(storedHealthy, "id = ?", healthyHandle.ID).Error)
	assert.Equal(t, entities.BuilderExecutorHandleStatusActive, storedHealthy.Status)

	storedDuplicate := &entities.BuilderExecutorHandle{}
	require.NoError(t, db.DB.First(storedDuplicate, "id = ?", duplicateHandle.ID).Error)
	assert.Equal(t, entities.BuilderExecutorHandleStatusTerminated, storedDuplicate.Status)
	require.NotNil(t, storedDuplicate.TerminatedAt)
}

func TestWriteAgentFilesRefreshesArtifactsFromWorkspaceRoot(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, run := seedBuilderWorkspaceServiceFixture(t)

	originalWriteBuilderWorkspaceFile := writeBuilderWorkspaceFile
	originalListBuilderWorkspaceFilesInContainer := listBuilderWorkspaceFilesInContainer
	t.Cleanup(func() {
		writeBuilderWorkspaceFile = originalWriteBuilderWorkspaceFile
		listBuilderWorkspaceFilesInContainer = originalListBuilderWorkspaceFilesInContainer
	})

	writtenPaths := make([]string, 0, 2)
	writeBuilderWorkspaceFile = func(_ *models.AppContext, podName, containerName, path, content string) error {
		assert.Equal(t, workspace.PodName, podName)
		assert.Equal(t, workspace.ContainerName, containerName)
		writtenPaths = append(writtenPaths, path+"="+content)
		return nil
	}
	listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ListFilesResponse, error) {
		assert.Equal(t, workspace.PodName, podName)
		assert.Equal(t, workspace.ContainerName, containerName)
		assert.Equal(t, workspace.WorkspaceRoot, path)
		return &models.ListFilesResponse{
			Path: workspace.WorkspaceRoot,
			Files: []models.FileInfo{
				{Name: "README.md", Type: "file", Size: 120},
				{Name: "package.json", Type: "file", Size: 80},
				{Name: "src", Type: "dir"},
			},
		}, nil
	}

	err := writeBuilderAgentFiles(context.Background(), workspace, run, []BuilderAgentFileWrite{
		{Path: "README.md", Content: "# Demo"},
		{Path: "package.json", Content: "{\"name\":\"demo\"}"},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"/workspace/README.md=# Demo",
		"/workspace/package.json={\"name\":\"demo\"}",
	}, writtenPaths)

	var artifacts []entities.BuilderArtifact
	require.NoError(t, db.DB.Where("session_id = ? AND workspace_id = ?", workspace.SessionID, workspace.ID).Order("path ASC").Find(&artifacts).Error)
	require.Len(t, artifacts, 2)
	assert.Equal(t, "workspace_file", string(artifacts[0].Kind))
	assert.Equal(t, "README.md", artifacts[0].Path)
	assert.Equal(t, run.ID, artifacts[0].RunID)
	assert.JSONEq(t, `{"size_bytes":120}`, artifacts[0].MetadataJSON)
	assert.Equal(t, "workspace_file", string(artifacts[1].Kind))
	assert.Equal(t, "package.json", artifacts[1].Path)
	assert.Equal(t, run.ID, artifacts[1].RunID)
	assert.JSONEq(t, `{"size_bytes":80}`, artifacts[1].MetadataJSON)
}

func TestCollectBuilderBuildArtifacts(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, run := seedBuilderWorkspaceServiceFixture(t)

	originalListBuilderWorkspaceFilesInContainer := listBuilderWorkspaceFilesInContainer
	t.Cleanup(func() {
		listBuilderWorkspaceFilesInContainer = originalListBuilderWorkspaceFilesInContainer
	})

	nestedListingPaths := make([]string, 0, 1)
	listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, filePath string) (*models.ListFilesResponse, error) {
		assert.Equal(t, workspace.PodName, podName)
		assert.Equal(t, workspace.ContainerName, containerName)
		nestedListingPaths = append(nestedListingPaths, filePath)
		if filePath == "/workspace/dist/assets" {
			return &models.ListFilesResponse{
				Path: filePath,
				Files: []models.FileInfo{
					{Name: "app.js", Type: "file", Size: 2048},
				},
			}, nil
		}
		return nil, errors.New("unexpected nested listing path")
	}

	artifacts, err := CollectBuilderBuildArtifacts(workspace, run, &models.ListFilesResponse{
		Path: "/workspace/dist",
		Files: []models.FileInfo{
			{Name: "index.html", Type: "file", Size: 512},
			{Name: "assets", Type: "dir"},
		},
	})
	require.NoError(t, err)
	require.Len(t, artifacts, 2)
	assert.Equal(t, []string{"/workspace/dist/assets"}, nestedListingPaths)
	assert.Equal(t, "build_output", string(artifacts[0].Kind))
	assert.Equal(t, "dist/index.html", artifacts[0].Path)
	assert.Equal(t, run.ID, artifacts[0].RunID)
	assert.JSONEq(t, `{"size_bytes":512,"output_root":"dist"}`, artifacts[0].MetadataJSON)
	assert.Equal(t, "build_output", string(artifacts[1].Kind))
	assert.Equal(t, "dist/assets/app.js", artifacts[1].Path)
	assert.Equal(t, run.ID, artifacts[1].RunID)
	assert.JSONEq(t, `{"size_bytes":2048,"output_root":"dist"}`, artifacts[1].MetadataJSON)
}

func TestCollectBuilderBuildArtifacts_SupportsNextOutputRoot(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, run := seedBuilderWorkspaceServiceFixture(t)

	originalListBuilderWorkspaceFilesInContainer := listBuilderWorkspaceFilesInContainer
	t.Cleanup(func() {
		listBuilderWorkspaceFilesInContainer = originalListBuilderWorkspaceFilesInContainer
	})

	nestedListingPaths := make([]string, 0, 1)
	listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, filePath string) (*models.ListFilesResponse, error) {
		assert.Equal(t, workspace.PodName, podName)
		assert.Equal(t, workspace.ContainerName, containerName)
		nestedListingPaths = append(nestedListingPaths, filePath)
		if filePath == "/workspace/.next/static" {
			return &models.ListFilesResponse{
				Path: filePath,
				Files: []models.FileInfo{
					{Name: "app.js", Type: "file", Size: 2048},
				},
			}, nil
		}
		return nil, errors.New("unexpected nested listing path")
	}

	artifacts, err := CollectBuilderBuildArtifacts(workspace, run, &models.ListFilesResponse{
		Path: "/workspace/.next",
		Files: []models.FileInfo{
			{Name: "routes-manifest.json", Type: "file", Size: 512},
			{Name: "static", Type: "dir"},
		},
	})
	require.NoError(t, err)
	require.Len(t, artifacts, 2)
	assert.Equal(t, []string{"/workspace/.next/static"}, nestedListingPaths)
	assert.Equal(t, "build_output", string(artifacts[0].Kind))
	assert.Equal(t, ".next/routes-manifest.json", artifacts[0].Path)
	assert.JSONEq(t, `{"size_bytes":512,"output_root":".next"}`, artifacts[0].MetadataJSON)
	assert.Equal(t, "build_output", string(artifacts[1].Kind))
	assert.Equal(t, ".next/static/app.js", artifacts[1].Path)
	assert.JSONEq(t, `{"size_bytes":2048,"output_root":".next"}`, artifacts[1].MetadataJSON)
}

func TestWriteAgentFilesRejectsLostOwnership(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, run := seedBuilderWorkspaceServiceFixture(t)

	claimToken := "workspace-owned-write-claim"
	claimedAt := time.Now().UTC().Add(-time.Minute)
	heartbeatAt := claimedAt.Add(10 * time.Second)
	timeoutAt := claimedAt.Add(time.Minute)
	run.ClaimToken = &claimToken
	run.ClaimedAt = &claimedAt
	run.HeartbeatAt = &heartbeatAt
	run.TimeoutAt = &timeoutAt
	require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"claim_token":  claimToken,
		"claimed_at":   claimedAt,
		"heartbeat_at": heartbeatAt,
		"timeout_at":   timeoutAt,
	}).Error)

	originalWriteBuilderWorkspaceFile := writeBuilderWorkspaceFile
	originalListBuilderWorkspaceFilesInContainer := listBuilderWorkspaceFilesInContainer
	t.Cleanup(func() {
		writeBuilderWorkspaceFile = originalWriteBuilderWorkspaceFile
		listBuilderWorkspaceFilesInContainer = originalListBuilderWorkspaceFilesInContainer
	})

	writeCount := 0
	writeBuilderWorkspaceFile = func(_ *models.AppContext, podName, containerName, path, content string) error {
		writeCount++
		return nil
	}
	listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ListFilesResponse, error) {
		return &models.ListFilesResponse{Path: workspace.WorkspaceRoot}, nil
	}

	require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status":       entities.BuilderRunStatusQueued,
		"phase":        entities.BuilderRunPhaseQueued,
		"claim_token":  nil,
		"claimed_at":   nil,
		"heartbeat_at": nil,
		"timeout_at":   nil,
	}).Error)

	err := writeBuilderAgentFiles(context.Background(), workspace, run, []BuilderAgentFileWrite{{Path: "README.md", Content: "# Demo"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Equal(t, 0, writeCount)
}

func TestDownloadBuilderWorkspaceStreamsWorkspaceRootContents(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, _ := seedBuilderWorkspaceServiceFixture(t)

	originalDownloadBuilderWorkspaceArchive := downloadBuilderWorkspaceArchive
	t.Cleanup(func() {
		downloadBuilderWorkspaceArchive = originalDownloadBuilderWorkspaceArchive
	})

	downloadBuilderWorkspaceArchive = func(_ *models.AppContext, podName, containerName, workspaceRoot string, writer io.Writer) error {
		assert.Equal(t, workspace.PodName, podName)
		assert.Equal(t, workspace.ContainerName, containerName)
		assert.Equal(t, workspace.WorkspaceRoot, workspaceRoot)
		_, err := writer.Write([]byte("archive-bytes"))
		return err
	}

	var buf bytes.Buffer
	err := DownloadBuilderWorkspace(context.Background(), "project-1", workspace.SessionID, &buf)
	require.NoError(t, err)
	assert.Equal(t, "archive-bytes", buf.String())
}

func TestListAndReadBuilderWorkspaceFilesRejectTraversalOutsideRoot(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, _ := seedBuilderWorkspaceServiceFixture(t)

	originalListBuilderWorkspaceFilesInContainer := listBuilderWorkspaceFilesInContainer
	originalReadBuilderWorkspaceFileInContainer := readBuilderWorkspaceFileInContainer
	t.Cleanup(func() {
		listBuilderWorkspaceFilesInContainer = originalListBuilderWorkspaceFilesInContainer
		readBuilderWorkspaceFileInContainer = originalReadBuilderWorkspaceFileInContainer
	})

	t.Run("list uses workspace root for safe relative path", func(t *testing.T) {
		listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ListFilesResponse, error) {
			assert.Equal(t, workspace.PodName, podName)
			assert.Equal(t, workspace.ContainerName, containerName)
			assert.Equal(t, "/workspace/src", path)
			return &models.ListFilesResponse{Path: path}, nil
		}

		result, err := ListBuilderWorkspaceFiles(context.Background(), "project-1", workspace.SessionID, "src")
		require.NoError(t, err)
		assert.Equal(t, "/workspace/src", result.Path)
	})

	t.Run("read uses workspace root for safe relative path", func(t *testing.T) {
		readBuilderWorkspaceFileInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ReadFileResponse, error) {
			assert.Equal(t, workspace.PodName, podName)
			assert.Equal(t, workspace.ContainerName, containerName)
			assert.Equal(t, "/workspace/README.md", path)
			return &models.ReadFileResponse{Path: path, Content: "# Demo", Size: 6}, nil
		}

		result, err := ReadBuilderWorkspaceFile(context.Background(), "project-1", workspace.SessionID, "README.md")
		require.NoError(t, err)
		assert.Equal(t, "/workspace/README.md", result.Path)
	})

	t.Run("list rejects traversal outside workspace root", func(t *testing.T) {
		called := false
		listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ListFilesResponse, error) {
			called = true
			return nil, nil
		}

		result, err := ListBuilderWorkspaceFiles(context.Background(), "project-1", workspace.SessionID, "../..")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.False(t, called)
	})

	t.Run("read rejects traversal outside workspace root", func(t *testing.T) {
		called := false
		readBuilderWorkspaceFileInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ReadFileResponse, error) {
			called = true
			return nil, nil
		}

		result, err := ReadBuilderWorkspaceFile(context.Background(), "project-1", workspace.SessionID, "../../secret.txt")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.False(t, called)
	})
}
