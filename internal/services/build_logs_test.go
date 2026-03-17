package services

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func setupBuildLogsServiceTestDB(t *testing.T) {
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
		&entities.Env{},
		&entities.Build{},
	))

	db.DB = testDB
}

func seedBuildLogsServiceFixture(t *testing.T, status entities.BuildStatus) *entities.Build {
	t.Helper()

	env := &entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "build-env",
		Name:             "Build Env",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "build-ns",
		IsBuildEnv:       true,
	}
	require.NoError(t, db.DB.Create(env).Error)

	build := &entities.Build{
		ID:               "build-1",
		CreatedAt:        time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
		BuildSettingID:   "setting-1",
		BuildNumber:      1,
		Status:           status,
		BuildEnvID:       env.ID,
		TriggerType:      entities.BuildTriggerManual,
		JobName:          "build-job-1",
		JobNamespace:     "build-ns",
		LogPersistStatus: entities.BuildLogPersistPending,
	}
	require.NoError(t, db.DB.Create(build).Error)

	return build
}

func buildLogsServicePod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "build-pod-1",
			Namespace: "build-ns",
			Labels: map[string]string{
				"job-name": "build-job-1",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{Name: "git-clone"},
			},
			Containers: []corev1.Container{
				{Name: "buildctl"},
			},
		},
	}
}

func TestStreamBuildLogs_StreamsActivePodLogs(t *testing.T) {
	setupBuildLogsServiceTestDB(t)
	build := seedBuildLogsServiceFixture(t, entities.BuildStatusBuilding)

	client := kubefake.NewSimpleClientset(buildLogsServicePod())

	originalGetBuildLogsClusterClient := getBuildLogsClusterClient
	originalOpenBuildLogsPodStream := openBuildLogsPodStream
	t.Cleanup(func() {
		getBuildLogsClusterClient = originalGetBuildLogsClusterClient
		openBuildLogsPodStream = originalOpenBuildLogsPodStream
	})

	getBuildLogsClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		require.Equal(t, "cluster-1", clusterID)
		return client, nil
	}
	openBuildLogsPodStream = func(
		_ context.Context,
		_ kubernetes.Interface,
		namespace, podName, containerName string,
		follow bool,
	) (io.ReadCloser, error) {
		require.Equal(t, "build-ns", namespace)
		require.Equal(t, "build-pod-1", podName)
		if containerName == "buildctl" {
			require.True(t, follow)
			return io.NopCloser(strings.NewReader("build-step\n")), nil
		}
		require.False(t, follow)
		return io.NopCloser(strings.NewReader("clone-step\n")), nil
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	StreamBuildLogs(c, build.ID)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "clone-step")
	assert.Contains(t, recorder.Body.String(), "build-step")
	assert.Contains(t, recorder.Body.String(), "stream ended")
}

func TestStreamBuildLogs_RetriesFollowContainerUntilLogsAreAvailable(t *testing.T) {
	setupBuildLogsServiceTestDB(t)
	build := seedBuildLogsServiceFixture(t, entities.BuildStatusBuilding)

	client := kubefake.NewSimpleClientset(buildLogsServicePod())

	originalGetBuildLogsClusterClient := getBuildLogsClusterClient
	originalOpenBuildLogsPodStream := openBuildLogsPodStream
	originalWaitForBuildLogsRetry := waitForBuildLogsRetry
	t.Cleanup(func() {
		getBuildLogsClusterClient = originalGetBuildLogsClusterClient
		openBuildLogsPodStream = originalOpenBuildLogsPodStream
		waitForBuildLogsRetry = originalWaitForBuildLogsRetry
	})

	buildctlAttempts := 0
	getBuildLogsClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		require.Equal(t, "cluster-1", clusterID)
		return client, nil
	}
	waitForBuildLogsRetry = func(ctx context.Context, delay time.Duration) error {
		return nil
	}
	openBuildLogsPodStream = func(
		_ context.Context,
		_ kubernetes.Interface,
		namespace, podName, containerName string,
		follow bool,
	) (io.ReadCloser, error) {
		require.Equal(t, "build-ns", namespace)
		require.Equal(t, "build-pod-1", podName)
		if containerName == "buildctl" {
			require.True(t, follow)
			buildctlAttempts += 1
			if buildctlAttempts == 1 {
				return nil, errors.New("container is waiting to start")
			}
			return io.NopCloser(strings.NewReader("build-step\n")), nil
		}
		require.False(t, follow)
		return io.NopCloser(strings.NewReader("clone-step\n")), nil
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	StreamBuildLogs(c, build.ID)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "clone-step")
	assert.Contains(t, recorder.Body.String(), "[buildctl] (logs not available yet)")
	assert.Contains(t, recorder.Body.String(), "build-step")
	assert.Contains(t, recorder.Body.String(), "stream ended")
	assert.Equal(t, 2, buildctlAttempts)
}

func TestStreamBuildLogs_StreamsArchivedLogsForTerminalBuild(t *testing.T) {
	setupBuildLogsServiceTestDB(t)
	build := seedBuildLogsServiceFixture(t, entities.BuildStatusSucceeded)
	build.LogPersistStatus = entities.BuildLogPersistSucceeded
	build.LogPath = "2026/03/build-1.log"
	require.NoError(t, db.DB.Save(build).Error)

	originalGetBuildLogsClusterClient := getBuildLogsClusterClient
	originalOpenPersistedBuildLog := openPersistedBuildLog
	t.Cleanup(func() {
		getBuildLogsClusterClient = originalGetBuildLogsClusterClient
		openPersistedBuildLog = originalOpenPersistedBuildLog
	})

	clusterClientCalled := false
	getBuildLogsClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		clusterClientCalled = true
		return nil, errors.New("should not be called")
	}
	openPersistedBuildLog = func(build *entities.Build) (io.ReadCloser, error) {
		require.Equal(t, "build-1", build.ID)
		return io.NopCloser(strings.NewReader("archived-log\n")), nil
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	StreamBuildLogs(c, build.ID)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "archived-log")
	assert.False(t, clusterClientCalled)
}

func TestStreamBuildLogs_ReturnsErrorWhenTerminalArchiveMissing(t *testing.T) {
	setupBuildLogsServiceTestDB(t)
	build := seedBuildLogsServiceFixture(t, entities.BuildStatusFailed)

	originalGetBuildLogsClusterClient := getBuildLogsClusterClient
	originalOpenPersistedBuildLog := openPersistedBuildLog
	t.Cleanup(func() {
		getBuildLogsClusterClient = originalGetBuildLogsClusterClient
		openPersistedBuildLog = originalOpenPersistedBuildLog
	})

	getBuildLogsClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		return kubefake.NewSimpleClientset(), nil
	}
	openPersistedBuildLog = func(build *entities.Build) (io.ReadCloser, error) {
		return nil, errors.New("archive unavailable")
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	StreamBuildLogs(c, build.ID)

	assert.Equal(t, 404, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "build log archive unavailable")
}

func TestCancelBuild_PersistsAvailableLogsBeforeDeletingJob(t *testing.T) {
	setupBuildLogsServiceTestDB(t)
	build := seedBuildLogsServiceFixture(t, entities.BuildStatusBuilding)

	originalPersistBuildLogs := persistBuildLogs
	originalCancelBuildJob := cancelBuildJob
	originalCleanupBuildSecrets := cleanupBuildSecrets
	t.Cleanup(func() {
		persistBuildLogs = originalPersistBuildLogs
		cancelBuildJob = originalCancelBuildJob
		cleanupBuildSecrets = originalCleanupBuildSecrets
	})

	order := make([]string, 0, 3)
	persistBuildLogs = func(ctx context.Context, buildID string) error {
		order = append(order, "persist")
		require.Equal(t, build.ID, buildID)
		return nil
	}
	cancelBuildJob = func(ctx context.Context, clusterID, jobName, jobNamespace string) error {
		order = append(order, "cancel")
		require.Equal(t, "cluster-1", clusterID)
		require.Equal(t, "build-job-1", jobName)
		require.Equal(t, "build-ns", jobNamespace)
		return nil
	}
	cleanupBuildSecrets = func(ctx context.Context, clusterID, buildID, namespace string) {
		order = append(order, "cleanup")
	}

	updatedBuild, err := CancelBuild(build.ID)
	require.NoError(t, err)

	assert.Equal(t, []string{"persist", "cancel"}, order[:2])
	assert.Equal(t, entities.BuildStatusCancelled, updatedBuild.Status)
	assert.Contains(t, order, "cleanup")
}
