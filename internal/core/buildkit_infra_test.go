package core

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildkitInfrastructureResources_UseDedicatedNamespace(t *testing.T) {
	resources := buildkitInfrastructureResources()

	if resources.Namespace.Name != "ketches-build" {
		t.Fatalf("expected buildkit namespace ketches-build, got %q", resources.Namespace.Name)
	}
	if resources.Service.Namespace != resources.Namespace.Name {
		t.Fatalf("expected service namespace %q, got %q", resources.Namespace.Name, resources.Service.Namespace)
	}
	if resources.StatefulSet.Namespace != resources.Namespace.Name {
		t.Fatalf("expected statefulset namespace %q, got %q", resources.Namespace.Name, resources.StatefulSet.Namespace)
	}
	if resources.BinfmtDaemonSet.Namespace != resources.Namespace.Name {
		t.Fatalf("expected daemonset namespace %q, got %q", resources.Namespace.Name, resources.BinfmtDaemonSet.Namespace)
	}
}

func TestBuildkitInfrastructureResources_CreateStatefulSetServiceAndBinfmtDaemonSet(t *testing.T) {
	resources := buildkitInfrastructureResources()

	if resources.Service.Name != "ketches-buildkitd" {
		t.Fatalf("unexpected buildkit service name %q", resources.Service.Name)
	}
	if len(resources.Service.Spec.Ports) != 1 || resources.Service.Spec.Ports[0].Port != 1234 {
		t.Fatalf("unexpected buildkit service ports %#v", resources.Service.Spec.Ports)
	}
	if resources.StatefulSet.Name != "ketches-buildkitd" {
		t.Fatalf("unexpected buildkit statefulset name %q", resources.StatefulSet.Name)
	}
	if resources.StatefulSet.Spec.Replicas == nil || *resources.StatefulSet.Spec.Replicas != 1 {
		t.Fatalf("expected one buildkitd replica, got %#v", resources.StatefulSet.Spec.Replicas)
	}
	if resources.BinfmtDaemonSet.Name != "ketches-buildkit-binfmt" {
		t.Fatalf("unexpected binfmt daemonset name %q", resources.BinfmtDaemonSet.Name)
	}
	if len(resources.BinfmtDaemonSet.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("expected one binfmt init container, got %#v", resources.BinfmtDaemonSet.Spec.Template.Spec.InitContainers)
	}
	if len(resources.BinfmtDaemonSet.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected one steady-state binfmt container, got %#v", resources.BinfmtDaemonSet.Spec.Template.Spec.Containers)
	}
	if resources.BinfmtDaemonSet.Spec.Template.Spec.InitContainers[0].SecurityContext == nil ||
		resources.BinfmtDaemonSet.Spec.Template.Spec.InitContainers[0].SecurityContext.Privileged == nil ||
		!*resources.BinfmtDaemonSet.Spec.Template.Spec.InitContainers[0].SecurityContext.Privileged {
		t.Fatalf("expected privileged binfmt installer, got %#v", resources.BinfmtDaemonSet.Spec.Template.Spec.InitContainers[0].SecurityContext)
	}
}

func TestBuildkitInfrastructureResources_UsePVCBackedWorkerState(t *testing.T) {
	resources := buildkitInfrastructureResources()

	if len(resources.StatefulSet.Spec.VolumeClaimTemplates) != 1 {
		t.Fatalf("expected one PVC template, got %#v", resources.StatefulSet.Spec.VolumeClaimTemplates)
	}
	if resources.StatefulSet.Spec.VolumeClaimTemplates[0].Name != "buildkit-state" {
		t.Fatalf("unexpected PVC template name %q", resources.StatefulSet.Spec.VolumeClaimTemplates[0].Name)
	}

	containers := resources.StatefulSet.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("expected one buildkitd container, got %#v", containers)
	}

	mountFound := false
	for _, mount := range containers[0].VolumeMounts {
		if mount.Name == "buildkit-state" && mount.MountPath == "/var/lib/buildkit" {
			mountFound = true
			break
		}
	}
	if !mountFound {
		t.Fatalf("expected /var/lib/buildkit mount, got %#v", containers[0].VolumeMounts)
	}
}

func TestBuildkitInfrastructureResources_RunBuildkitdPrivileged(t *testing.T) {
	resources := buildkitInfrastructureResources()

	containers := resources.StatefulSet.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("expected one buildkitd container, got %#v", containers)
	}
	if containers[0].SecurityContext == nil || containers[0].SecurityContext.Privileged == nil || !*containers[0].SecurityContext.Privileged {
		t.Fatalf("expected privileged buildkitd security context, got %#v", containers[0].SecurityContext)
	}
}

func TestValidateBuildkitStatefulSetReadiness_ReturnsClearMessageWhenBuilderIsNotReady(t *testing.T) {
	err := validateBuildkitStatefulSetReadiness(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildkitStatefulSetName,
			Namespace: buildkitNamespaceName,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: int32Ptr(1),
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 0,
		},
	})
	if err == nil {
		t.Fatalf("expected builder readiness error")
	}
	expected := "BuildKit builder is not ready in namespace ketches-build. Wait for StatefulSet ketches-buildkitd to become Ready before retrying."
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidateBinfmtDaemonSetReadiness_SkipsSinglePlatformBuilds(t *testing.T) {
	err := validateBinfmtDaemonSetReadiness("linux/amd64", nil)
	if err != nil {
		t.Fatalf("expected single-platform build to skip binfmt check, got %v", err)
	}
}

func TestValidateBinfmtDaemonSetReadiness_ReturnsClearMessageWhenBinfmtIsNotReady(t *testing.T) {
	err := validateBinfmtDaemonSetReadiness("linux/amd64,linux/arm64", &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildkitBinfmtDaemonSetName,
			Namespace: buildkitNamespaceName,
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			NumberReady:            1,
		},
	})
	if err == nil {
		t.Fatalf("expected binfmt readiness error")
	}
	expected := "Multi-arch build requires binfmt/QEMU support, but DaemonSet ketches-buildkit-binfmt is not ready in namespace ketches-build."
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestBuildkitNodeName_ReturnsReadyBuilderPodNode(t *testing.T) {
	nodeName, err := buildkitNodeName([]corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ketches-buildkitd-0"},
			Spec:       corev1.PodSpec{NodeName: "node-arm64-1"},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected ready buildkit pod node, got error %v", err)
	}
	if nodeName != "node-arm64-1" {
		t.Fatalf("expected node-arm64-1, got %q", nodeName)
	}
}

func TestValidateBinfmtPodReadinessOnNode_ReturnsClearMessageWhenBuilderNodePodIsNotReady(t *testing.T) {
	err := validateBinfmtPodReadinessOnNode("linux/amd64,linux/arm64", "node-arm64-1", []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ketches-buildkit-binfmt-abc"},
			Spec:       corev1.PodSpec{NodeName: "node-arm64-1"},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionFalse},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ketches-buildkit-binfmt-other"},
			Spec:       corev1.PodSpec{NodeName: "node-amd64-1"},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected node-level binfmt readiness error")
	}
	expected := "Multi-arch build requires binfmt/QEMU support on builder node node-arm64-1, but Pod for DaemonSet ketches-buildkit-binfmt is not Ready in namespace ketches-build."
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidateBinfmtPodReadinessOnNode_AllowsReadyPodOnBuilderNode(t *testing.T) {
	err := validateBinfmtPodReadinessOnNode("linux/amd64,linux/arm64", "node-amd64-1", []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ketches-buildkit-binfmt-abc"},
			Spec:       corev1.PodSpec{NodeName: "node-arm64-1"},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ketches-buildkit-binfmt-def"},
			Spec:       corev1.PodSpec{NodeName: "node-amd64-1"},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected ready binfmt pod on builder node, got %v", err)
	}
}

func TestApplyBuildkitService_PreservesExistingClusterIPAndResourceVersion(t *testing.T) {
	client := &fakeServiceClient{
		existing: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:            buildkitServiceName,
				Namespace:       buildkitNamespaceName,
				ResourceVersion: "7",
			},
			Spec: corev1.ServiceSpec{
				ClusterIP:  "10.96.0.10",
				ClusterIPs: []string{"10.96.0.10"},
			},
		},
	}

	if err := applyBuildkitService(context.Background(), client, buildkitInfrastructureResources().Service); err != nil {
		t.Fatalf("applyBuildkitService returned error: %v", err)
	}

	if client.updated == nil {
		t.Fatalf("expected update to be called")
	}
	if client.updated.ResourceVersion != "7" {
		t.Fatalf("expected resourceVersion 7, got %q", client.updated.ResourceVersion)
	}
	if client.updated.Spec.ClusterIP != "10.96.0.10" {
		t.Fatalf("expected clusterIP to be preserved, got %q", client.updated.Spec.ClusterIP)
	}
}

func TestApplyBuildkitStatefulSet_UsesExistingResourceVersion(t *testing.T) {
	client := &fakeStatefulSetClient{
		existing: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:            buildkitStatefulSetName,
				Namespace:       buildkitNamespaceName,
				ResourceVersion: "11",
			},
		},
	}

	if err := applyBuildkitStatefulSet(context.Background(), client, buildkitInfrastructureResources().StatefulSet); err != nil {
		t.Fatalf("applyBuildkitStatefulSet returned error: %v", err)
	}

	if client.updated == nil {
		t.Fatalf("expected update to be called")
	}
	if client.updated.ResourceVersion != "11" {
		t.Fatalf("expected resourceVersion 11, got %q", client.updated.ResourceVersion)
	}
}

type fakeServiceClient struct {
	existing *corev1.Service
	updated  *corev1.Service
}

func (f *fakeServiceClient) Get(context.Context, string, metav1.GetOptions) (*corev1.Service, error) {
	return f.existing, nil
}

func (f *fakeServiceClient) Create(context.Context, *corev1.Service, metav1.CreateOptions) (*corev1.Service, error) {
	return nil, nil
}

func (f *fakeServiceClient) Update(_ context.Context, service *corev1.Service, _ metav1.UpdateOptions) (*corev1.Service, error) {
	f.updated = service
	return service, nil
}

type fakeStatefulSetClient struct {
	existing *appsv1.StatefulSet
	updated  *appsv1.StatefulSet
}

func (f *fakeStatefulSetClient) Get(context.Context, string, metav1.GetOptions) (*appsv1.StatefulSet, error) {
	return f.existing, nil
}

func (f *fakeStatefulSetClient) Create(context.Context, *appsv1.StatefulSet, metav1.CreateOptions) (*appsv1.StatefulSet, error) {
	return nil, nil
}

func (f *fakeStatefulSetClient) Update(_ context.Context, statefulSet *appsv1.StatefulSet, _ metav1.UpdateOptions) (*appsv1.StatefulSet, error) {
	f.updated = statefulSet
	return statefulSet, nil
}

func int32Ptr(v int32) *int32 {
	return &v
}
