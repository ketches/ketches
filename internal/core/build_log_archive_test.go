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

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

type tempFileCheckingReadCloser struct {
	reader  *strings.Reader
	onFirst func()
	checked bool
}

func (r *tempFileCheckingReadCloser) Read(p []byte) (int, error) {
	if !r.checked {
		r.checked = true
		if r.onFirst != nil {
			r.onFirst()
		}
	}
	return r.reader.Read(p)
}

func (r *tempFileCheckingReadCloser) Close() error {
	return nil
}

func setupBuildLogArchiveTestDB(t *testing.T) {
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

func setupBuildLogArchiveConfig(t *testing.T) string {
	t.Helper()

	originalConfig := app.Config
	baseDir := t.TempDir()
	app.Config = app.AppConfig{
		BuildLogBaseDir:       baseDir,
		BuildLogRetentionDays: 15,
	}
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	return baseDir
}

func setupBuildLogArchiveClientHooks(
	t *testing.T,
	client kubernetes.Interface,
	logs map[string]string,
	logErrs map[string]error,
) {
	t.Helper()

	originalGetClusterClient := getBuildLogClusterClient
	originalOpenPodLogStream := openBuildPodLogStream

	getBuildLogClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		if clusterID != "cluster-1" {
			return nil, errors.New("unexpected cluster")
		}
		return client, nil
	}
	openBuildPodLogStream = func(
		_ context.Context,
		_ kubernetes.Interface,
		namespace, podName, containerName string,
	) (io.ReadCloser, error) {
		require.Equal(t, "build-ns", namespace)
		require.Equal(t, "build-pod-1", podName)
		if err, ok := logErrs[containerName]; ok {
			return nil, err
		}
		return io.NopCloser(strings.NewReader(logs[containerName])), nil
	}

	t.Cleanup(func() {
		getBuildLogClusterClient = originalGetClusterClient
		openBuildPodLogStream = originalOpenPodLogStream
	})
}

func seedBuildLogArchiveFixture(t *testing.T, status entities.BuildStatus) *entities.Build {
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

func buildLogArchivePod() *corev1.Pod {
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

func TestPersistBuildLogs_WritesContainersInDisplayOrder(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)
	build := seedBuildLogArchiveFixture(t, entities.BuildStatusSucceeded)

	client := kubefake.NewSimpleClientset(buildLogArchivePod())
	setupBuildLogArchiveClientHooks(t, client, map[string]string{
		"git-clone": "clone-step\n",
		"buildctl":  "build-step\n",
	}, map[string]error{})

	err := PersistBuildLogs(context.Background(), build.ID)
	require.NoError(t, err)

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSucceeded, updated.LogPersistStatus)
	assert.NotEmpty(t, updated.LogPath)
	assert.NotNil(t, updated.LogPersistedAt)
	assert.NotNil(t, updated.LogExpireAt)
	assert.Equal(t, int64(len("clone-step\nbuild-step\n")), updated.LogSize)

	data, err := os.ReadFile(filepath.Join(baseDir, updated.LogPath))
	require.NoError(t, err)
	assert.Equal(t, "clone-step\nbuild-step\n", string(data))
}

func TestPersistBuildLogs_IsIdempotentWhenArchiveAlreadyExists(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)
	build := seedBuildLogArchiveFixture(t, entities.BuildStatusSucceeded)

	build.LogPath = filepath.Join("2026", "03", build.ID+".log")
	build.LogPersistStatus = entities.BuildLogPersistSucceeded
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "2026", "03"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, build.LogPath), []byte("already-archived\n"), 0o644))
	require.NoError(t, db.DB.Save(build).Error)

	client := kubefake.NewSimpleClientset(buildLogArchivePod())
	callCount := 0
	setupBuildLogArchiveClientHooks(t, client, map[string]string{}, map[string]error{})
	originalOpenPodLogStream := openBuildPodLogStream
	openBuildPodLogStream = func(
		_ context.Context,
		_ kubernetes.Interface,
		_, _, _ string,
	) (io.ReadCloser, error) {
		callCount++
		return originalOpenPodLogStream(context.Background(), client, "build-ns", "build-pod-1", "buildctl")
	}

	err := PersistBuildLogs(context.Background(), build.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, callCount)

	data, err := os.ReadFile(filepath.Join(baseDir, build.LogPath))
	require.NoError(t, err)
	assert.Equal(t, "already-archived\n", string(data))
}

func TestPersistBuildLogs_UsesTemporaryFileThenRenames(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)
	build := seedBuildLogArchiveFixture(t, entities.BuildStatusSucceeded)

	client := kubefake.NewSimpleClientset(buildLogArchivePod())
	setupBuildLogArchiveClientHooks(t, client, map[string]string{
		"git-clone": "clone-step\n",
		"buildctl":  "build-step\n",
	}, map[string]error{})

	err := PersistBuildLogs(context.Background(), build.ID)
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(baseDir, "tmp", "*.tmp"))
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestPersistBuildLogs_StreamsContainersIntoArchiveFileInDisplayOrder(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)
	build := seedBuildLogArchiveFixture(t, entities.BuildStatusSucceeded)

	client := kubefake.NewSimpleClientset(buildLogArchivePod())
	setupBuildLogArchiveClientHooks(t, client, map[string]string{
		"git-clone": "clone-step\n",
		"buildctl":  "build-step\n",
	}, map[string]error{})

	err := PersistBuildLogs(context.Background(), build.ID)
	require.NoError(t, err)

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)

	data, err := os.ReadFile(filepath.Join(baseDir, updated.LogPath))
	require.NoError(t, err)
	assert.Equal(t, "clone-step\nbuild-step\n", string(data))

	matches, err := filepath.Glob(filepath.Join(baseDir, "tmp", "*.tmp"))
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestPersistBuildLogs_CreatesTempArchiveBeforeStreamingLogs(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	baseDir := setupBuildLogArchiveConfig(t)
	build := seedBuildLogArchiveFixture(t, entities.BuildStatusSucceeded)
	client := kubefake.NewSimpleClientset(buildLogArchivePod())

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

	tmpPath := filepath.Join(baseDir, "tmp", build.ID+".log.tmp")
	openBuildPodLogStream = func(
		_ context.Context,
		_ kubernetes.Interface,
		namespace, podName, containerName string,
	) (io.ReadCloser, error) {
		require.Equal(t, "build-ns", namespace)
		require.Equal(t, "build-pod-1", podName)
		return &tempFileCheckingReadCloser{
			reader: strings.NewReader(containerName + "\n"),
			onFirst: func() {
				_, err := os.Stat(tmpPath)
				require.NoError(t, err)
			},
		}, nil
	}

	err := PersistBuildLogs(context.Background(), build.ID)
	require.NoError(t, err)
}

func TestPersistBuildLogs_MarksSourceUnavailableWhenNoBuildPodExists(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	setupBuildLogArchiveConfig(t)
	build := seedBuildLogArchiveFixture(t, entities.BuildStatusFailed)

	client := kubefake.NewSimpleClientset()
	setupBuildLogArchiveClientHooks(t, client, map[string]string{}, map[string]error{})

	err := PersistBuildLogs(context.Background(), build.ID)
	require.Error(t, err)

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSourceUnavailable, updated.LogPersistStatus)
	assert.Empty(t, updated.LogPath)
	assert.NotEmpty(t, updated.LogPersistError)
	assert.Nil(t, updated.LogPersistedAt)
}

func TestPersistBuildLogs_MarksSourceUnavailableWhenNoContainerLogCanBeOpened(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	setupBuildLogArchiveConfig(t)
	build := seedBuildLogArchiveFixture(t, entities.BuildStatusFailed)

	client := kubefake.NewSimpleClientset(buildLogArchivePod())
	setupBuildLogArchiveClientHooks(t, client, map[string]string{}, map[string]error{
		"git-clone": errors.New("log source not found"),
		"buildctl":  errors.New("log source not found"),
	})

	err := PersistBuildLogs(context.Background(), build.ID)
	require.Error(t, err)

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildLogPersistSourceUnavailable, updated.LogPersistStatus)
	assert.Empty(t, updated.LogPath)
	assert.NotEmpty(t, updated.LogPersistError)
	assert.Nil(t, updated.LogPersistedAt)
}

func TestPersistBuildLogs_MarksRetryableFailureWhenClusterClientUnavailable(t *testing.T) {
	setupBuildLogArchiveTestDB(t)
	setupBuildLogArchiveConfig(t)
	build := seedBuildLogArchiveFixture(t, entities.BuildStatusFailed)

	originalGetClusterClient := getBuildLogClusterClient
	t.Cleanup(func() {
		getBuildLogClusterClient = originalGetClusterClient
	})

	getBuildLogClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		require.Equal(t, "cluster-1", clusterID)
		return nil, errors.New("cluster unavailable")
	}

	err := PersistBuildLogs(context.Background(), build.ID)
	require.Error(t, err)

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildLogPersistFailed, updated.LogPersistStatus)
	assert.Empty(t, updated.LogPath)
	assert.Contains(t, updated.LogPersistError, "cluster unavailable")
	assert.Nil(t, updated.LogPersistedAt)
}

func TestOpenPersistedBuildLog_FailsWhenArchiveMissing(t *testing.T) {
	setupBuildLogArchiveConfig(t)

	build := &entities.Build{
		ID:               "build-1",
		LogPath:          filepath.Join("2026", "03", "build-1.log"),
		LogPersistStatus: entities.BuildLogPersistSucceeded,
	}

	stream, err := OpenPersistedBuildLog(build)
	require.Error(t, err)
	assert.Nil(t, stream)
}
