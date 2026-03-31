package core

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildBuilderWorkspacePVC_UsesStableDeterministicSessionScopedName(t *testing.T) {
	spec := testBuilderWorkspaceSpec()

	pvcOne, err := BuildBuilderWorkspacePVC(spec)
	if err != nil {
		t.Fatalf("BuildBuilderWorkspacePVC returned error: %v", err)
	}
	pvcTwo, err := BuildBuilderWorkspacePVC(spec)
	if err != nil {
		t.Fatalf("BuildBuilderWorkspacePVC returned error: %v", err)
	}

	if pvcOne.Name != "builder-workspace-session-1" {
		t.Fatalf("expected deterministic PVC name %q, got %q", "builder-workspace-session-1", pvcOne.Name)
	}
	if pvcTwo.Name != pvcOne.Name {
		t.Fatalf("expected stable PVC name %q, got %q", pvcOne.Name, pvcTwo.Name)
	}
	if pvcOne.Namespace != spec.Namespace {
		t.Fatalf("expected PVC namespace %q, got %q", spec.Namespace, pvcOne.Namespace)
	}
	if pvcOne.Labels[builderWorkspaceMarkerLabelKey] != "true" {
		t.Fatalf("expected builder marker label on PVC, got %#v", pvcOne.Labels)
	}
	if pvcOne.Labels[builderSessionIDLabelKey] != spec.SessionID {
		t.Fatalf("expected builder session label %q, got %#v", spec.SessionID, pvcOne.Labels)
	}
	storageRequest, ok := pvcOne.Spec.Resources.Requests[corev1.ResourceStorage]
	if !ok {
		t.Fatalf("expected PVC storage request, got %#v", pvcOne.Spec.Resources.Requests)
	}
	if storageRequest.Cmp(resource.MustParse(spec.StorageRequest)) != 0 {
		t.Fatalf("expected PVC storage request %q, got %q", spec.StorageRequest, storageRequest.String())
	}
}

func TestBuildBuilderWorkspacePod_UsesBuildEnvNamespaceSessionPVCMountImageAndLabels(t *testing.T) {
	setBuilderWorkspaceConfigForTest(t)
	spec := testBuilderWorkspaceSpec()

	pod, err := BuildBuilderWorkspacePod(spec)
	if err != nil {
		t.Fatalf("BuildBuilderWorkspacePod returned error: %v", err)
	}

	if pod.Namespace != spec.Namespace {
		t.Fatalf("expected pod namespace %q, got %q", spec.Namespace, pod.Namespace)
	}
	if len(pod.Spec.Volumes) != 1 {
		t.Fatalf("expected one workspace volume, got %#v", pod.Spec.Volumes)
	}
	if pod.Spec.Volumes[0].PersistentVolumeClaim == nil {
		t.Fatalf("expected workspace volume to use PVC, got %#v", pod.Spec.Volumes[0])
	}
	pvc, err := BuildBuilderWorkspacePVC(spec)
	if err != nil {
		t.Fatalf("BuildBuilderWorkspacePVC returned error: %v", err)
	}
	if pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName != pvc.Name {
		t.Fatalf("expected pod to mount PVC %q, got %#v", pvc.Name, pod.Spec.Volumes[0].PersistentVolumeClaim)
	}
	if len(pod.Spec.Containers) != 1 {
		t.Fatalf("expected one main workspace container, got %#v", pod.Spec.Containers)
	}

	container := pod.Spec.Containers[0]
	if container.Name != builderWorkspaceContainerName {
		t.Fatalf("expected workspace container name %q, got %q", builderWorkspaceContainerName, container.Name)
	}
	if container.Image != app.Config.BuilderWorkspaceImage {
		t.Fatalf("expected workspace image %q, got %q", app.Config.BuilderWorkspaceImage, container.Image)
	}
	if len(container.Command) != 3 {
		t.Fatalf("expected long-running workspace command, got %#v", container.Command)
	}
	if container.Command[0] != "sh" || container.Command[1] != "-lc" {
		t.Fatalf("expected shell-based workspace command, got %#v", container.Command)
	}
	if container.Command[2] != "while true; do sleep 3600; done" {
		t.Fatalf("expected keepalive workspace command, got %#v", container.Command)
	}
	if !hasBuilderWorkspaceMount(container.VolumeMounts, builderWorkspaceVolumeName, app.Config.BuilderWorkspaceRoot) {
		t.Fatalf("expected workspace mount %q on %q, got %#v", builderWorkspaceVolumeName, app.Config.BuilderWorkspaceRoot, container.VolumeMounts)
	}
	if pod.Labels[kube.LabelProjectID] != spec.ProjectID {
		t.Fatalf("expected project ID label %q, got %#v", spec.ProjectID, pod.Labels)
	}
	if pod.Labels[kube.LabelProjectSlug] != spec.ProjectSlug {
		t.Fatalf("expected project slug label %q, got %#v", spec.ProjectSlug, pod.Labels)
	}
	if pod.Labels[kube.LabelEnvID] != spec.BuildEnvID {
		t.Fatalf("expected env ID label %q, got %#v", spec.BuildEnvID, pod.Labels)
	}
	if pod.Labels[kube.LabelEnvSlug] != spec.BuildEnvSlug {
		t.Fatalf("expected env slug label %q, got %#v", spec.BuildEnvSlug, pod.Labels)
	}
	if pod.Labels[kube.LabelManagedBy] != "true" {
		t.Fatalf("expected managed-by label on pod, got %#v", pod.Labels)
	}
	if pod.Labels[builderWorkspaceMarkerLabelKey] != "true" {
		t.Fatalf("expected builder marker label on pod, got %#v", pod.Labels)
	}
	if pod.Labels[builderSessionIDLabelKey] != spec.SessionID {
		t.Fatalf("expected builder session label %q, got %#v", spec.SessionID, pod.Labels)
	}
}

func TestBuildBuilderWorkspacePod_UsesExecutionImageOverrideWhenProvided(t *testing.T) {
	setBuilderWorkspaceConfigForTest(t)
	spec := testBuilderWorkspaceSpec()
	spec.ExecutionImage = "ghcr.io/ketches/builder-node-static:2026-03-29"

	pod, err := BuildBuilderWorkspacePod(spec)
	if err != nil {
		t.Fatalf("BuildBuilderWorkspacePod returned error: %v", err)
	}

	if got := pod.Spec.Containers[0].Image; got != "ghcr.io/ketches/builder-node-static:2026-03-29" {
		t.Fatalf("expected execution image override %q, got %q", "ghcr.io/ketches/builder-node-static:2026-03-29", got)
	}
}

func TestBuildBuilderWorkspaceResources_RejectsMissingWorkspaceConfigAndInvalidSpec(t *testing.T) {
	t.Run("rejects missing workspace image", func(t *testing.T) {
		original := app.Config
		app.Config = app.AppConfig{BuilderWorkspaceRoot: "/builder-workspace"}
		t.Cleanup(func() {
			app.Config = original
		})

		_, err := BuildBuilderWorkspaceResources(testBuilderWorkspaceSpec())
		if err == nil {
			t.Fatalf("expected missing workspace image error")
		}
	})

	t.Run("rejects invalid session id for resource names", func(t *testing.T) {
		setBuilderWorkspaceConfigForTest(t)
		spec := testBuilderWorkspaceSpec()
		spec.SessionID = "Session With Spaces"

		_, err := BuildBuilderWorkspaceResources(spec)
		if err == nil {
			t.Fatalf("expected invalid session id error")
		}
	})
}

func TestBuildBuilderWorkspaceResources_ReturnsPVCAndPodWithoutServiceInPhaseOne(t *testing.T) {
	setBuilderWorkspaceConfigForTest(t)
	spec := testBuilderWorkspaceSpec()

	resources, err := BuildBuilderWorkspaceResources(spec)
	if err != nil {
		t.Fatalf("BuildBuilderWorkspaceResources returned error: %v", err)
	}

	if resources.PersistentVolumeClaim == nil {
		t.Fatalf("expected workspace PVC resource")
	}
	if resources.Pod == nil {
		t.Fatalf("expected workspace Pod resource")
	}
	if resources.Service != nil {
		t.Fatalf("expected no Service resource in phase one, got %#v", resources.Service)
	}
	pvc, err := BuildBuilderWorkspacePVC(spec)
	if err != nil {
		t.Fatalf("BuildBuilderWorkspacePVC returned error: %v", err)
	}
	if resources.PersistentVolumeClaim.Name != pvc.Name {
		t.Fatalf("expected bundled PVC name %q, got %q", pvc.Name, resources.PersistentVolumeClaim.Name)
	}
	pod, err := BuildBuilderWorkspacePod(spec)
	if err != nil {
		t.Fatalf("BuildBuilderWorkspacePod returned error: %v", err)
	}
	if resources.Pod.Name != pod.Name {
		t.Fatalf("expected bundled Pod name %q, got %q", pod.Name, resources.Pod.Name)
	}
}

func hasBuilderWorkspaceMount(mounts []corev1.VolumeMount, name, mountPath string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == mountPath {
			return true
		}
	}

	return false
}

func setBuilderWorkspaceConfigForTest(t *testing.T) {
	t.Helper()
	original := app.Config
	app.Config = app.AppConfig{
		BuilderWorkspaceImage: "ghcr.io/ketches/builder-workspace:latest",
		BuilderWorkspaceRoot:  "/builder-workspace",
	}
	t.Cleanup(func() {
		app.Config = original
	})
}

func testBuilderWorkspaceSpec() BuilderWorkspaceSpec {
	return BuilderWorkspaceSpec{
		SessionID:      "session-1",
		ProjectID:      "project-1",
		ProjectSlug:    "demo",
		BuildEnvID:     "env-1",
		BuildEnvSlug:   "dev",
		Namespace:      "project-dev",
		StorageRequest: "5Gi",
	}
}
