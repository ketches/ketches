package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ketches/ketches/internal/kube"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	nodeTerminalNamespace               = "default"
	nodeTerminalContainerName           = "shell"
	nodeTerminalLabelKey                = "ketches.io/node-terminal"
	nodeTerminalLabelValue              = "true"
	nodeTerminalNodeNameLabelKey        = "ketches.io/node-name"
	nodeTerminalLastActiveAnnotationKey = "ketches.io/last-active-at"
	nodeTerminalIdleTimeout             = 30 * time.Minute
	nodeTerminalCleanupInterval         = 10 * time.Minute
	nodeTerminalStartupTimeout          = 30 * time.Second
	nodeTerminalNamePrefix              = "node-terminal-"
)

var nodeTerminalBrokenWaitingReasons = map[string]struct{}{
	"CrashLoopBackOff":           {},
	"CreateContainerConfigError": {},
	"CreateContainerError":       {},
	"ErrImagePull":               {},
	"ImagePullBackOff":           {},
	"RunContainerError":          {},
}

var (
	nodeTerminalCleanupOnce  = &sync.Once{}
	nodeTerminalCleanupDone  <-chan struct{}
	nodeTerminalNow          = func() time.Time { return time.Now().UTC() }
	nodeTerminalWaitInterval = time.Second
)

func buildNodeTerminalPodName(nodeName string) string {
	sanitized := strings.ToLower(nodeName)
	sanitized = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '.':
			return r
		default:
			return '-'
		}
	}, sanitized)
	sanitized = strings.Trim(sanitized, "-.")
	if sanitized == "" {
		sanitized = "node"
	}

	maxLen := 253 - len(nodeTerminalNamePrefix)
	if len(sanitized) > maxLen {
		sanitized = strings.TrimRight(sanitized[:maxLen], "-.")
		if sanitized == "" {
			sanitized = "node"
		}
	}

	return nodeTerminalNamePrefix + sanitized
}

func buildNodeTerminalPod(nodeName, podName string, now time.Time) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: nodeTerminalNamespace,
			Labels: map[string]string{
				nodeTerminalLabelKey:         nodeTerminalLabelValue,
				nodeTerminalNodeNameLabelKey: nodeName,
			},
			Annotations: map[string]string{
				nodeTerminalLastActiveAnnotationKey: now.UTC().Format(time.RFC3339),
			},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Containers: []corev1.Container{
				{
					Name:    nodeTerminalContainerName,
					Image:   "alpine:latest",
					Command: []string{"sh", "-c", "while true; do sleep 3600; done"},
					SecurityContext: &corev1.SecurityContext{
						Privileged: boolPtr(true),
					},
				},
			},
			HostPID:       true,
			HostNetwork:   true,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

func isReusableNodeTerminalPod(pod *corev1.Pod, nodeName string) bool {
	return !shouldRecreateNodeTerminalPod(pod, nodeName)
}

func shouldRecreateNodeTerminalPod(pod *corev1.Pod, nodeName string) bool {
	if pod == nil {
		return true
	}
	if pod.DeletionTimestamp != nil {
		return true
	}
	if pod.Labels[nodeTerminalLabelKey] != nodeTerminalLabelValue {
		return true
	}
	if pod.Labels[nodeTerminalNodeNameLabelKey] != nodeName {
		return true
	}
	if pod.Spec.NodeName != nodeName {
		return true
	}
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
		return true
	}

	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != nodeTerminalContainerName || status.State.Waiting == nil {
			continue
		}
		if _, broken := nodeTerminalBrokenWaitingReasons[status.State.Waiting.Reason]; broken {
			return true
		}
	}

	return false
}

func isNodeTerminalPodExpired(pod *corev1.Pod, now time.Time) bool {
	if pod == nil {
		return true
	}

	lastActiveAt, ok := pod.Annotations[nodeTerminalLastActiveAnnotationKey]
	if !ok {
		return true
	}

	parsed, err := time.Parse(time.RFC3339, lastActiveAt)
	if err != nil {
		return true
	}

	return !parsed.Add(nodeTerminalIdleTimeout).After(now)
}

func ensureNodeTerminalPod(ctx context.Context, pods corev1client.PodInterface, nodeName string, now time.Time) (*corev1.Pod, error) {
	podName := buildNodeTerminalPodName(nodeName)
	pod, err := pods.Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("get node terminal pod %s: %w", podName, err)
		}

		pod, err = createNodeTerminalPod(ctx, pods, nodeName, podName, now)
		if err != nil {
			return nil, err
		}
	} else if shouldRecreateNodeTerminalPod(pod, nodeName) {
		if err := deleteNodeTerminalPod(ctx, pods, pod.Name); err != nil {
			return nil, err
		}

		pod, err = createNodeTerminalPod(ctx, pods, nodeName, podName, now)
		if err != nil {
			return nil, err
		}
	}

	return touchNodeTerminalPodActivity(ctx, pods, pod, now)
}

func createNodeTerminalPod(ctx context.Context, pods corev1client.PodInterface, nodeName, podName string, now time.Time) (*corev1.Pod, error) {
	pod, err := pods.Create(ctx, buildNodeTerminalPod(nodeName, podName, now), metav1.CreateOptions{})
	if err == nil {
		return pod, nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("create node terminal pod %s: %w", podName, err)
	}

	pod, getErr := pods.Get(ctx, podName, metav1.GetOptions{})
	if getErr != nil {
		return nil, fmt.Errorf("get node terminal pod %s after create race: %w", podName, getErr)
	}

	return pod, nil
}

func touchNodeTerminalPodActivity(ctx context.Context, pods corev1client.PodInterface, pod *corev1.Pod, now time.Time) (*corev1.Pod, error) {
	if pod == nil {
		return nil, fmt.Errorf("node terminal pod is nil")
	}

	updated := pod.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	updated.Annotations[nodeTerminalLastActiveAnnotationKey] = now.UTC().Format(time.RFC3339)

	updatedPod, err := pods.Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("update node terminal pod %s activity: %w", updated.Name, err)
	}

	return updatedPod, nil
}

func waitForNodeTerminalPodRunning(ctx context.Context, pods corev1client.PodInterface, nodeName, podName string) (*corev1.Pod, error) {
	ticker := time.NewTicker(nodeTerminalWaitInterval)
	defer ticker.Stop()

	for {
		pod, err := pods.Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("node terminal pod %s disappeared before becoming ready: %w", podName, ctx.Err())
				case <-ticker.C:
					continue
				}
			}
			return nil, fmt.Errorf("get node terminal pod %s while waiting for readiness: %w", podName, err)
		}

		if shouldRecreateNodeTerminalPod(pod, nodeName) {
			return nil, fmt.Errorf("node terminal pod %s became unhealthy before it was ready", podName)
		}
		if pod.Status.Phase == corev1.PodRunning {
			return pod, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("node terminal pod %s did not become ready within %s: %w", podName, nodeTerminalStartupTimeout, ctx.Err())
		case <-ticker.C:
		}
	}
}

func deleteNodeTerminalPod(ctx context.Context, pods corev1client.PodInterface, podName string) error {
	gracePeriodSeconds := int64(0)
	err := pods.Delete(ctx, podName, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete node terminal pod %s: %w", podName, err)
	}
	return nil
}

func cleanupIdleNodeTerminalPods(ctx context.Context, pods corev1client.PodInterface, now time.Time) error {
	podList, err := pods.List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", nodeTerminalLabelKey, nodeTerminalLabelValue),
	})
	if err != nil {
		return fmt.Errorf("list node terminal pods: %w", err)
	}

	for _, item := range podList.Items {
		pod := item.DeepCopy()
		expectedNodeName := pod.Labels[nodeTerminalNodeNameLabelKey]
		if expectedNodeName == "" {
			expectedNodeName = pod.Spec.NodeName
		}

		if !isNodeTerminalPodExpired(pod, now) && !shouldRecreateNodeTerminalPod(pod, expectedNodeName) {
			continue
		}

		if err := deleteNodeTerminalPod(ctx, pods, pod.Name); err != nil {
			return err
		}
	}

	return nil
}

func StartClusterNodeTerminalCleanupLoop(ctx context.Context) <-chan struct{} {
	nodeTerminalCleanupOnce.Do(func() {
		done := make(chan struct{})
		nodeTerminalCleanupDone = done
		go func() {
			defer close(done)
			runClusterNodeTerminalCleanupLoop(ctx, nodeTerminalCleanupInterval, nodeTerminalNow, cleanupClusterNodeTerminalPodsAcrossClusters)
		}()
	})

	return nodeTerminalCleanupDone
}

func runClusterNodeTerminalCleanupLoop(
	ctx context.Context,
	interval time.Duration,
	now func() time.Time,
	cleanup func(time.Time),
) {
	if ctx == nil {
		ctx = context.Background()
	}

	cleanup(now())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup(now())
		}
	}
}

func cleanupClusterNodeTerminalPodsAcrossClusters(now time.Time) {
	clusters, err := ListClustersSimple()
	if err != nil {
		log.Printf("node terminal cleanup: failed to list clusters: %v", err)
		return
	}

	for _, cluster := range clusters {
		if !cluster.Enabled {
			continue
		}

		client, err := kube.GlobalClusterStore.GetClient(cluster.ID)
		if err != nil {
			log.Printf("node terminal cleanup: failed to get cluster client %s: %v", cluster.ID, err)
			continue
		}

		if err := cleanupIdleNodeTerminalPods(context.Background(), client.CoreV1().Pods(nodeTerminalNamespace), now); err != nil {
			log.Printf("node terminal cleanup: failed for cluster %s: %v", cluster.ID, err)
		}
	}
}
