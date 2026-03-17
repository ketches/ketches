package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var getBuildLogsClusterClient = func(clusterID string) (kubernetes.Interface, error) {
	return kube.GlobalClusterStore.GetClient(clusterID)
}

var openBuildLogsPodStream = func(
	ctx context.Context,
	client kubernetes.Interface,
	namespace, podName, containerName string,
	follow bool,
) (io.ReadCloser, error) {
	tailLines := int64(1000)
	return client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     follow,
		TailLines:  &tailLines,
		Timestamps: true,
	}).Stream(ctx)
}

var openPersistedBuildLog = core.OpenPersistedBuildLog
var persistBuildLogs = core.PersistBuildLogs
var cancelBuildJob = core.CancelBuildJob
var cleanupBuildSecrets = core.CleanupBuildSecrets
var waitForBuildLogsRetry = func(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

const buildLogRetryDelay = time.Second

func StreamBuildLogs(c *gin.Context, buildID string) {
	build, err := GetBuild(buildID)
	if err != nil {
		c.JSON(404, gin.H{"error": "build not found"})
		return
	}

	switch build.Status {
	case entities.BuildStatusPending, entities.BuildStatusCloning, entities.BuildStatusBuilding:
		streamActiveBuildLogs(c, build)
	default:
		if err := streamPersistedBuildLogs(c, build); err == nil {
			return
		}
		if err := streamActiveBuildLogs(c, build); err == nil {
			return
		}
		c.JSON(404, gin.H{"error": "build log archive unavailable"})
	}
}

func streamPersistedBuildLogs(c *gin.Context, build *entities.Build) error {
	stream, err := openPersistedBuildLog(build)
	if err != nil {
		return err
	}
	defer stream.Close()

	writeBuildLogSSEHeaders(c)
	if err := streamSSELogReader(c, stream); err != nil {
		return err
	}
	c.SSEvent("done", "stream ended")
	c.Writer.Flush()
	return nil
}

func streamActiveBuildLogs(c *gin.Context, build *entities.Build) error {
	if build.JobName == "" {
		return errors.New("build has no job")
	}

	requestCtx := buildLogsRequestContext(c)

	buildEnv, err := GetEnv(build.BuildEnvID)
	if err != nil {
		return fmt.Errorf("failed to load build environment: %w", err)
	}
	client, err := getBuildLogsClusterClient(buildEnv.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster client: %w", err)
	}

	pods, err := client.CoreV1().Pods(build.JobNamespace).List(requestCtx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", build.JobName),
	})
	if err != nil || len(pods.Items) == 0 {
		return errors.New("build pod not found")
	}

	pod := &pods.Items[0]
	containerNames := make([]string, 0, len(pod.Spec.InitContainers)+len(pod.Spec.Containers))
	for _, container := range pod.Spec.InitContainers {
		containerNames = append(containerNames, container.Name)
	}
	for _, container := range pod.Spec.Containers {
		containerNames = append(containerNames, container.Name)
	}

	follow := build.Status == entities.BuildStatusPending ||
		build.Status == entities.BuildStatusCloning ||
		build.Status == entities.BuildStatusBuilding

	writeBuildLogSSEHeaders(c)

	for i, containerName := range containerNames {
		containerFollow := follow && i == len(containerNames)-1
		stream, err := openBuildLogsPodStream(requestCtx, client, build.JobNamespace, pod.Name, containerName, containerFollow)
		if err != nil {
			c.SSEvent("log", fmt.Sprintf("[%s] (logs not available yet)\n", containerName))
			c.Writer.Flush()
			if !containerFollow {
				continue
			}

			stream, err = retryOpenFollowBuildLogsPodStream(
				requestCtx,
				client,
				build.JobNamespace,
				pod.Name,
				containerName,
			)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				continue
			}
		}

		if err := streamSSELogReader(c, stream); err != nil {
			log.Printf("Build log stream error %s: %v", containerName, err)
		}
		stream.Close()
	}

	c.SSEvent("done", "stream ended")
	c.Writer.Flush()
	return nil
}

func writeBuildLogSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
}

func buildLogsRequestContext(c *gin.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}

	return context.Background()
}

func retryOpenFollowBuildLogsPodStream(
	ctx context.Context,
	client kubernetes.Interface,
	namespace, podName, containerName string,
) (io.ReadCloser, error) {
	for {
		if err := waitForBuildLogsRetry(ctx, buildLogRetryDelay); err != nil {
			return nil, err
		}

		stream, err := openBuildLogsPodStream(ctx, client, namespace, podName, containerName, true)
		if err == nil {
			return stream, nil
		}
	}
}

func streamSSELogReader(c *gin.Context, stream io.Reader) error {
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			c.SSEvent("log", string(buf[:n]))
			c.Writer.Flush()
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
