package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var errBuildLogSourceUnavailable = errors.New("build pod logs unavailable")

var getBuildLogClusterClient = func(clusterID string) (kubernetes.Interface, error) {
	return kube.GlobalClusterStore.GetClient(clusterID)
}

var openBuildPodLogStream = func(
	ctx context.Context,
	client kubernetes.Interface,
	namespace, podName, containerName string,
) (io.ReadCloser, error) {
	return client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:  containerName,
		Timestamps: true,
	}).Stream(ctx)
}

func PersistBuildLogs(ctx context.Context, buildID string) error {
	var build entities.Build
	if err := db.DB.First(&build, "id = ?", buildID).Error; err != nil {
		return err
	}

	if BuildHasPersistedLog(&build) {
		if _, err := os.Stat(buildLogArchiveAbsPath(build.LogPath)); err == nil {
			return nil
		}
	}

	var buildEnv entities.Env
	if err := db.DB.First(&buildEnv, "id = ?", build.BuildEnvID).Error; err != nil {
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, app.WrapErrorf(err, "failed to load build environment: %w", err))
	}

	client, err := getBuildLogClusterClient(buildEnv.ClusterID)
	if err != nil {
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, app.WrapErrorf(err, "failed to get cluster client: %w", err))
	}

	pods, err := client.CoreV1().Pods(build.JobNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", build.JobName),
	})
	if err != nil {
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, app.WrapErrorf(err, "failed to list build pods: %w", err))
	}
	if len(pods.Items) == 0 {
		return persistBuildLogFailure(&build, entities.BuildLogPersistSourceUnavailable, errBuildLogSourceUnavailable)
	}

	relPath, absPath, tmpPath, err := buildLogArchivePaths(&build)
	if err != nil {
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, err)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, app.WrapErrorf(err, "failed to create archive directory: %w", err))
	}
	if err := os.MkdirAll(filepath.Dir(tmpPath), 0o755); err != nil {
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, app.WrapErrorf(err, "failed to create temporary archive directory: %w", err))
	}
	_ = os.Remove(tmpPath)

	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, app.WrapErrorf(err, "failed to create temporary archive file: %w", err))
	}

	logSize, err := writeBuildPodLogsToArchive(ctx, client, build.JobNamespace, &pods.Items[0], tmpFile)
	closeErr := tmpFile.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		status := entities.BuildLogPersistFailed
		if errors.Is(err, errBuildLogSourceUnavailable) {
			status = entities.BuildLogPersistSourceUnavailable
		}
		return persistBuildLogFailure(&build, status, err)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, app.WrapErrorf(closeErr, "failed to close temporary archive file: %w", closeErr))
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		return persistBuildLogFailure(&build, entities.BuildLogPersistFailed, app.WrapErrorf(err, "failed to finalize archive file: %w", err))
	}

	now := time.Now()
	expireAt := now.Add(time.Duration(buildLogRetentionDays()) * 24 * time.Hour)
	build.LogPath = relPath
	build.LogSize = logSize
	build.LogPersistStatus = entities.BuildLogPersistSucceeded
	build.LogPersistError = ""
	build.LogPersistedAt = &now
	build.LogExpireAt = &expireAt

	return db.DB.Save(&build).Error
}

func BuildHasPersistedLog(build *entities.Build) bool {
	return build != nil &&
		build.LogPersistStatus == entities.BuildLogPersistSucceeded &&
		strings.TrimSpace(build.LogPath) != ""
}

func OpenPersistedBuildLog(build *entities.Build) (io.ReadCloser, error) {
	if !BuildHasPersistedLog(build) {
		return nil, os.ErrNotExist
	}
	return os.Open(buildLogArchiveAbsPath(build.LogPath))
}

func writeBuildPodLogsToArchive(
	ctx context.Context,
	client kubernetes.Interface,
	namespace string,
	pod *corev1.Pod,
	dst io.Writer,
) (int64, error) {
	containerNames := make([]string, 0, len(pod.Spec.InitContainers)+len(pod.Spec.Containers))
	for _, container := range pod.Spec.InitContainers {
		containerNames = append(containerNames, container.Name)
	}
	for _, container := range pod.Spec.Containers {
		containerNames = append(containerNames, container.Name)
	}

	openedAnyStream := false
	var totalWritten int64

	for _, containerName := range containerNames {
		stream, err := openBuildPodLogStream(ctx, client, namespace, pod.Name, containerName)
		if err != nil {
			continue
		}

		openedAnyStream = true
		written, copyErr := io.Copy(dst, stream)
		stream.Close()
		if copyErr != nil {
			return 0, app.WrapErrorf(copyErr, "failed to copy logs for container %s: %w", containerName, copyErr)
		}
		totalWritten += written
	}

	if !openedAnyStream {
		return 0, errBuildLogSourceUnavailable
	}

	return totalWritten, nil
}

func buildLogArchivePaths(build *entities.Build) (string, string, string, error) {
	if build == nil {
		return "", "", "", errors.New("build is required")
	}

	baseDir := buildLogBaseDir()
	if strings.TrimSpace(baseDir) == "" {
		return "", "", "", errors.New("build log base dir is not configured")
	}

	createdAt := build.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	relPath := filepath.Join(createdAt.Format("2006"), createdAt.Format("01"), build.ID+".log")
	absPath := buildLogArchiveAbsPath(relPath)
	tmpPath := filepath.Join(baseDir, "tmp", build.ID+".log.tmp")
	return relPath, absPath, tmpPath, nil
}

func buildLogArchiveAbsPath(relPath string) string {
	return filepath.Join(buildLogBaseDir(), relPath)
}

func buildLogBaseDir() string {
	if strings.TrimSpace(app.Config.BuildLogBaseDir) == "" {
		return "data/build-logs"
	}
	return app.Config.BuildLogBaseDir
}

func buildLogRetentionDays() int {
	if app.Config.BuildLogRetentionDays <= 0 {
		return 15
	}
	return app.Config.BuildLogRetentionDays
}

func persistBuildLogFailure(build *entities.Build, status entities.BuildLogPersistStatus, err error) error {
	build.LogPath = ""
	build.LogSize = 0
	build.LogPersistStatus = status
	build.LogPersistError = err.Error()
	build.LogPersistedAt = nil
	build.LogExpireAt = nil
	if saveErr := db.DB.Save(build).Error; saveErr != nil {
		return saveErr
	}
	return err
}
