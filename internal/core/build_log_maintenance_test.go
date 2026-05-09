package core

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func seedMaintenanceBuild(
	t *testing.T,
	id string,
	status entities.BuildStatus,
	logStatus entities.BuildLogPersistStatus,
	jobName string,
	expireAt *time.Time,
) *entities.Build {
	t.Helper()

	build := &entities.Build{
		ID:               id,
		CreatedAt:        time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
		BuildSettingID:   "setting-" + id,
		BuildNumber:      1,
		Status:           status,
		BuildEnvID:       "env-1",
		TriggerType:      entities.BuildTriggerManual,
		JobName:          jobName,
		JobNamespace:     "build-ns",
		LogPersistStatus: logStatus,
		LogExpireAt:      expireAt,
	}
	require.NoError(t, db.DB.Create(build).Error)
	return build
}

func TestRecoverTerminalBuildLogArchives_RetriesPendingAndFailedBuildsOnly(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)

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

	succeededPending := seedMaintenanceBuild(t, "build-pending", entities.BuildStatusSucceeded, entities.BuildLogPersistPending, "job-pending", nil)
	failedFailed := seedMaintenanceBuild(t, "build-failed", entities.BuildStatusFailed, entities.BuildLogPersistFailed, "job-failed", nil)
	sourceUnavailable := seedMaintenanceBuild(t, "build-source-unavailable", entities.BuildStatusSucceeded, entities.BuildLogPersistSourceUnavailable, "job-source-unavailable", nil)
	expiredBuild := seedMaintenanceBuild(t, "build-expired", entities.BuildStatusSucceeded, entities.BuildLogPersistExpired, "job-expired", nil)
	activeBuild := seedMaintenanceBuild(t, "build-active", entities.BuildStatusBuilding, entities.BuildLogPersistPending, "job-active", nil)

	client := kubefake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-pending",
				Namespace: "build-ns",
				Labels: map[string]string{
					"job-name": "job-pending",
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "git-clone"}},
				Containers:     []corev1.Container{{Name: "buildctl"}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-failed",
				Namespace: "build-ns",
				Labels: map[string]string{
					"job-name": "job-failed",
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "git-clone"}},
				Containers:     []corev1.Container{{Name: "buildctl"}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-source-unavailable",
				Namespace: "build-ns",
				Labels: map[string]string{
					"job-name": "job-source-unavailable",
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "git-clone"}},
				Containers:     []corev1.Container{{Name: "buildctl"}},
			},
		},
	)

	originalGetClusterClient := getBuildLogClusterClient
	originalOpenPodLogStream := openBuildPodLogStream
	t.Cleanup(func() {
		getBuildLogClusterClient = originalGetClusterClient
		openBuildPodLogStream = originalOpenPodLogStream
	})

	getBuildLogClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		require.Equal(t, "cluster-1", clusterID)
		return client, nil
	}
	openBuildPodLogStream = func(
		_ context.Context,
		_ kubernetes.Interface,
		_,
		podName,
		containerName string,
	) (io.ReadCloser, error) {
		switch podName {
		case "pod-pending":
			return io.NopCloser(strings.NewReader("pending-" + containerName + "\n")), nil
		case "pod-failed":
			return io.NopCloser(strings.NewReader("failed-" + containerName + "\n")), nil
		case "pod-source-unavailable":
			return io.NopCloser(strings.NewReader("source-unavailable-" + containerName + "\n")), nil
		default:
			return nil, errors.New("unexpected pod")
		}
	}

	err := RecoverTerminalBuildLogArchives(context.Background())
	require.NoError(t, err)

	var updatedPending entities.Build
	require.NoError(t, db.DB.First(&updatedPending, "id = ?", succeededPending.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSucceeded, updatedPending.LogPersistStatus)
	assert.FileExists(t, filepath.Join(baseDir, updatedPending.LogPath))

	var updatedFailed entities.Build
	require.NoError(t, db.DB.First(&updatedFailed, "id = ?", failedFailed.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSucceeded, updatedFailed.LogPersistStatus)
	assert.FileExists(t, filepath.Join(baseDir, updatedFailed.LogPath))

	var unchangedExpired entities.Build
	require.NoError(t, db.DB.First(&unchangedExpired, "id = ?", expiredBuild.ID).Error)
	assert.Equal(t, entities.BuildLogPersistExpired, unchangedExpired.LogPersistStatus)

	var unchangedSourceUnavailable entities.Build
	require.NoError(t, db.DB.First(&unchangedSourceUnavailable, "id = ?", sourceUnavailable.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSourceUnavailable, unchangedSourceUnavailable.LogPersistStatus)
	assert.Empty(t, unchangedSourceUnavailable.LogPath)

	var unchangedActive entities.Build
	require.NoError(t, db.DB.First(&unchangedActive, "id = ?", activeBuild.ID).Error)
	assert.Equal(t, entities.BuildLogPersistPending, unchangedActive.LogPersistStatus)
	assert.Empty(t, unchangedActive.LogPath)
}

func TestRecoverTerminalBuildLogArchives_SkipsSourceUnavailableBuilds(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	setupBuildLogArchiveConfig(t)

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

	sourceUnavailable := seedMaintenanceBuild(
		t,
		"build-source-unavailable",
		entities.BuildStatusSucceeded,
		entities.BuildLogPersistSourceUnavailable,
		"job-source-unavailable",
		nil,
	)

	client := kubefake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-source-unavailable",
				Namespace: "build-ns",
				Labels: map[string]string{
					"job-name": "job-source-unavailable",
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "git-clone"}},
				Containers:     []corev1.Container{{Name: "buildctl"}},
			},
		},
	)

	originalGetClusterClient := getBuildLogClusterClient
	originalOpenPodLogStream := openBuildPodLogStream
	t.Cleanup(func() {
		getBuildLogClusterClient = originalGetClusterClient
		openBuildPodLogStream = originalOpenPodLogStream
	})

	getBuildLogClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		require.Equal(t, "cluster-1", clusterID)
		return client, nil
	}
	openBuildPodLogStream = func(
		_ context.Context,
		_ kubernetes.Interface,
		_,
		podName,
		containerName string,
	) (io.ReadCloser, error) {
		require.Equal(t, "pod-source-unavailable", podName)
		return io.NopCloser(strings.NewReader("source-unavailable-" + containerName + "\n")), nil
	}

	err := RecoverTerminalBuildLogArchives(context.Background())
	require.NoError(t, err)

	var unchanged entities.Build
	require.NoError(t, db.DB.First(&unchanged, "id = ?", sourceUnavailable.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSourceUnavailable, unchanged.LogPersistStatus)
	assert.Empty(t, unchanged.LogPath)
}

func TestRecoverTerminalBuildLogArchives_StopsWhenContextIsCancelled(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	setupBuildLogArchiveConfig(t)

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

	firstBuild := seedMaintenanceBuild(t, "build-first", entities.BuildStatusSucceeded, entities.BuildLogPersistPending, "job-first", nil)
	secondBuild := seedMaintenanceBuild(t, "build-second", entities.BuildStatusSucceeded, entities.BuildLogPersistPending, "job-second", nil)

	client := kubefake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-first",
				Namespace: "build-ns",
				Labels: map[string]string{
					"job-name": "job-first",
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "git-clone"}},
				Containers:     []corev1.Container{{Name: "buildctl"}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-second",
				Namespace: "build-ns",
				Labels: map[string]string{
					"job-name": "job-second",
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "git-clone"}},
				Containers:     []corev1.Container{{Name: "buildctl"}},
			},
		},
	)

	originalGetClusterClient := getBuildLogClusterClient
	originalOpenPodLogStream := openBuildPodLogStream
	t.Cleanup(func() {
		getBuildLogClusterClient = originalGetClusterClient
		openBuildPodLogStream = originalOpenPodLogStream
	})

	ctx, cancel := context.WithCancel(context.Background())
	getBuildLogClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		require.Equal(t, "cluster-1", clusterID)
		return client, nil
	}
	openBuildPodLogStream = func(
		_ context.Context,
		_ kubernetes.Interface,
		_,
		podName,
		containerName string,
	) (io.ReadCloser, error) {
		if podName == "pod-first" {
			cancel()
		}
		return io.NopCloser(strings.NewReader(podName + "-" + containerName + "\n")), nil
	}

	err := RecoverTerminalBuildLogArchives(ctx)
	require.NoError(t, err)

	var updatedFirst entities.Build
	require.NoError(t, db.DB.First(&updatedFirst, "id = ?", firstBuild.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSucceeded, updatedFirst.LogPersistStatus)

	var unchangedSecond entities.Build
	require.NoError(t, db.DB.First(&unchangedSecond, "id = ?", secondBuild.ID).Error)
	assert.Equal(t, entities.BuildLogPersistPending, unchangedSecond.LogPersistStatus)
	assert.Empty(t, unchangedSecond.LogPath)
}

func TestDeleteExpiredBuildLogs_RemovesArchiveAndMarksMetadataExpired(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)

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

	expireAt := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	build := seedMaintenanceBuild(t, "build-expire-me", entities.BuildStatusSucceeded, entities.BuildLogPersistSucceeded, "job-expire", &expireAt)
	build.LogPath = filepath.Join("2026", "03", build.ID+".log")
	build.LogSize = 12
	require.NoError(t, db.DB.Save(build).Error)

	archivePath := filepath.Join(baseDir, build.LogPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
	require.NoError(t, os.WriteFile(archivePath, []byte("archived-log"), 0o644))

	err := DeleteExpiredBuildLogs(context.Background(), time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildLogPersistExpired, updated.LogPersistStatus)
	assert.Empty(t, updated.LogPath)
	assert.Empty(t, updated.LogPersistError)
	assert.Equal(t, int64(0), updated.LogSize)
	assert.NoFileExists(t, archivePath)
}

func TestDeleteExpiredBuildLogs_LeavesActiveArchivesUntouched(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)

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

	expireAt := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	build := seedMaintenanceBuild(t, "build-keep", entities.BuildStatusSucceeded, entities.BuildLogPersistSucceeded, "job-keep", &expireAt)
	build.LogPath = filepath.Join("2026", "03", build.ID+".log")
	build.LogSize = 12
	require.NoError(t, db.DB.Save(build).Error)

	archivePath := filepath.Join(baseDir, build.LogPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
	require.NoError(t, os.WriteFile(archivePath, []byte("archived-log"), 0o644))

	err := DeleteExpiredBuildLogs(context.Background(), time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSucceeded, updated.LogPersistStatus)
	assert.Equal(t, build.LogPath, updated.LogPath)
	assert.FileExists(t, archivePath)
}

func TestStartBuildLogMaintenance_RunsCleanupImmediately(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)

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

	expireAt := time.Now().UTC().Add(-time.Hour)
	build := seedMaintenanceBuild(t, "build-expire-now", entities.BuildStatusSucceeded, entities.BuildLogPersistSucceeded, "job-expire-now", &expireAt)
	build.LogPath = filepath.Join("2026", "03", build.ID+".log")
	build.LogSize = 12
	require.NoError(t, db.DB.Save(build).Error)

	archivePath := filepath.Join(baseDir, build.LogPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
	require.NoError(t, os.WriteFile(archivePath, []byte("archived-log"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		StartBuildLogMaintenance(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildLogPersistExpired, updated.LogPersistStatus)
	assert.Empty(t, updated.LogPath)
	assert.NoFileExists(t, archivePath)
}
