package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

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

	buildEnv, err := GetEnv(build.BuildEnvID)
	if err != nil {
		return fmt.Errorf("failed to load build environment: %w", err)
	}
	client, err := getBuildLogsClusterClient(buildEnv.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster client: %w", err)
	}

	pods, err := client.CoreV1().Pods(build.JobNamespace).List(context.Background(), metav1.ListOptions{
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
		stream, err := openBuildLogsPodStream(context.Background(), client, build.JobNamespace, pod.Name, containerName, containerFollow)
		if err != nil {
			c.SSEvent("log", fmt.Sprintf("[%s] (logs not available yet)\n", containerName))
			c.Writer.Flush()
			continue
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
