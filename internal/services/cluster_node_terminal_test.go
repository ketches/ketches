package services

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestBuildNodeTerminalPodNameSanitizesNodeName(t *testing.T) {
	got := buildNodeTerminalPodName("Node_A/worker.1")

	assert.Equal(t, "node-terminal-node-a-worker.1", got)
}

func TestBuildNodeTerminalPodAddsExpectedLabelsAndAnnotation(t *testing.T) {
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	podName := buildNodeTerminalPodName("worker-a")

	pod := buildNodeTerminalPod("worker-a", podName, now)
	require.NotNil(t, pod)
	require.Len(t, pod.Spec.Containers, 1)
	require.NotNil(t, pod.Spec.Containers[0].SecurityContext)
	require.NotNil(t, pod.Spec.Containers[0].SecurityContext.Privileged)

	assert.Equal(t, podName, pod.Name)
	assert.Equal(t, nodeTerminalNamespace, pod.Namespace)
	assert.Equal(t, "worker-a", pod.Spec.NodeName)
	assert.Equal(t, nodeTerminalLabelValue, pod.Labels[nodeTerminalLabelKey])
	assert.Equal(t, "worker-a", pod.Labels[nodeTerminalNodeNameLabelKey])
	assert.Equal(t, now.Format(time.RFC3339), pod.Annotations[nodeTerminalLastActiveAnnotationKey])
	assert.Equal(t, nodeTerminalContainerName, pod.Spec.Containers[0].Name)
	assert.Equal(t, "alpine:latest", pod.Spec.Containers[0].Image)
	assert.True(t, *pod.Spec.Containers[0].SecurityContext.Privileged)
	assert.True(t, pod.Spec.HostPID)
	assert.True(t, pod.Spec.HostNetwork)
	assert.Equal(t, corev1.RestartPolicyNever, pod.Spec.RestartPolicy)
}

func TestIsReusableNodeTerminalPodAcceptsHealthyMatchingPod(t *testing.T) {
	pod := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), time.Now().UTC())
	pod.Status.Phase = corev1.PodRunning

	assert.True(t, isReusableNodeTerminalPod(pod, "worker-a"))
}

func TestShouldRecreateNodeTerminalPodRejectsTerminalOrBrokenPods(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "failed pod",
			pod: func() *corev1.Pod {
				pod := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), now)
				pod.Status.Phase = corev1.PodFailed
				return pod
			}(),
			want: true,
		},
		{
			name: "wrong node",
			pod: func() *corev1.Pod {
				pod := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), now)
				pod.Status.Phase = corev1.PodRunning
				pod.Spec.NodeName = "worker-b"
				return pod
			}(),
			want: true,
		},
		{
			name: "image pull failure",
			pod: func() *corev1.Pod {
				pod := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), now)
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: nodeTerminalContainerName,
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{Reason: "ErrImagePull"},
						},
					},
				}
				return pod
			}(),
			want: true,
		},
		{
			name: "healthy running pod",
			pod: func() *corev1.Pod {
				pod := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), now)
				pod.Status.Phase = corev1.PodRunning
				return pod
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldRecreateNodeTerminalPod(tt.pod, "worker-a"))
		})
	}
}

func TestIsNodeTerminalPodExpiredUsesLastActiveAt(t *testing.T) {
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)

	expiredPod := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), now.Add(-nodeTerminalIdleTimeout-time.Minute))
	freshPod := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), now.Add(-10*time.Minute))
	missingAnnotationPod := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), now)
	delete(missingAnnotationPod.Annotations, nodeTerminalLastActiveAnnotationKey)

	assert.True(t, isNodeTerminalPodExpired(expiredPod, now))
	assert.False(t, isNodeTerminalPodExpired(freshPod, now))
	assert.True(t, isNodeTerminalPodExpired(missingAnnotationPod, now))
}

func TestEnsureNodeTerminalPodCreatesWhenMissing(t *testing.T) {
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	client := fake.NewSimpleClientset()

	pod, err := ensureNodeTerminalPod(context.Background(), client.CoreV1().Pods(nodeTerminalNamespace), "worker-a", now)
	require.NoError(t, err)

	stored, err := client.CoreV1().Pods(nodeTerminalNamespace).Get(t.Context(), buildNodeTerminalPodName("worker-a"), metav1.GetOptions{})
	require.NoError(t, err)

	assert.Equal(t, stored.Name, pod.Name)
	assert.Equal(t, now.Format(time.RFC3339), stored.Annotations[nodeTerminalLastActiveAnnotationKey])
}

func TestEnsureNodeTerminalPodRecoversFromAlreadyExistsRace(t *testing.T) {
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	client := fake.NewSimpleClientset()
	podResource := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

	client.PrependReactor("create", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		createAction := action.(ktesting.CreateAction)
		pod := createAction.GetObject().(*corev1.Pod).DeepCopy()
		if err := client.Tracker().Add(pod); err != nil {
			return true, nil, err
		}
		return true, nil, apierrors.NewAlreadyExists(podResource.GroupResource(), pod.Name)
	})

	pod, err := ensureNodeTerminalPod(context.Background(), client.CoreV1().Pods(nodeTerminalNamespace), "worker-a", now)
	require.NoError(t, err)

	assert.Equal(t, buildNodeTerminalPodName("worker-a"), pod.Name)
}

func TestEnsureNodeTerminalPodReusesExistingPodAndTouchesActivity(t *testing.T) {
	oldTime := time.Date(2026, time.March, 15, 11, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	existing := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), oldTime)
	existing.Status.Phase = corev1.PodRunning

	client := fake.NewSimpleClientset(existing)

	pod, err := ensureNodeTerminalPod(context.Background(), client.CoreV1().Pods(nodeTerminalNamespace), "worker-a", now)
	require.NoError(t, err)

	assert.Equal(t, now.Format(time.RFC3339), pod.Annotations[nodeTerminalLastActiveAnnotationKey])
}

func TestEnsureNodeTerminalPodDeletesAndRecreatesBrokenPod(t *testing.T) {
	oldTime := time.Date(2026, time.March, 15, 11, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	existing := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), oldTime)
	existing.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			Name: nodeTerminalContainerName,
			State: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{Reason: "ErrImagePull"},
			},
		},
	}

	client := fake.NewSimpleClientset(existing)

	pod, err := ensureNodeTerminalPod(context.Background(), client.CoreV1().Pods(nodeTerminalNamespace), "worker-a", now)
	require.NoError(t, err)

	assert.Equal(t, now.Format(time.RFC3339), pod.Annotations[nodeTerminalLastActiveAnnotationKey])
	assert.Empty(t, pod.Status.ContainerStatuses)
}

func TestCleanupIdleNodeTerminalPodsDeletesExpiredAndBrokenPods(t *testing.T) {
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	expired := buildNodeTerminalPod("worker-a", buildNodeTerminalPodName("worker-a"), now.Add(-nodeTerminalIdleTimeout-time.Minute))
	broken := buildNodeTerminalPod("worker-b", buildNodeTerminalPodName("worker-b"), now)
	broken.Status.Phase = corev1.PodFailed
	fresh := buildNodeTerminalPod("worker-c", buildNodeTerminalPodName("worker-c"), now.Add(-5*time.Minute))
	unrelated := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "plain-pod",
			Namespace: nodeTerminalNamespace,
		},
	}

	client := fake.NewSimpleClientset(expired, broken, fresh, unrelated)

	err := cleanupIdleNodeTerminalPods(context.Background(), client.CoreV1().Pods(nodeTerminalNamespace), now)
	require.NoError(t, err)

	_, expiredErr := client.CoreV1().Pods(nodeTerminalNamespace).Get(t.Context(), expired.Name, metav1.GetOptions{})
	_, brokenErr := client.CoreV1().Pods(nodeTerminalNamespace).Get(t.Context(), broken.Name, metav1.GetOptions{})
	freshPod, freshErr := client.CoreV1().Pods(nodeTerminalNamespace).Get(t.Context(), fresh.Name, metav1.GetOptions{})
	unrelatedPod, unrelatedErr := client.CoreV1().Pods(nodeTerminalNamespace).Get(t.Context(), unrelated.Name, metav1.GetOptions{})

	assert.True(t, apierrors.IsNotFound(expiredErr))
	assert.True(t, apierrors.IsNotFound(brokenErr))
	require.NoError(t, freshErr)
	require.NoError(t, unrelatedErr)
	assert.Equal(t, fresh.Name, freshPod.Name)
	assert.Equal(t, unrelated.Name, unrelatedPod.Name)
}

func TestRunClusterNodeTerminalCleanupLoopStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{}, 1)
	done := make(chan struct{})
	var calls atomic.Int32

	go func() {
		defer close(done)
		runClusterNodeTerminalCleanupLoop(ctx, time.Hour, func() time.Time {
			return time.Date(2026, time.March, 17, 12, 0, 0, 0, time.UTC)
		}, func(time.Time) {
			if calls.Add(1) == 1 {
				started <- struct{}{}
			}
		})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("cleanup loop did not run immediately")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup loop did not stop after context cancellation")
	}
}
