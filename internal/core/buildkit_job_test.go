package core

import (
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateBuildClientJob_RunsInEnvNamespace(t *testing.T) {
	job, err := CreateBuildClientJobFromCodeRepo(
		&entities.Build{ID: "build-1", BuildNumber: 7, GitRef: "main", ImageFullName: "demo/api:v1.2.3"},
		&entities.CodeRepository{Name: "demo-api", GitRepoURL: "https://github.com/example/demo-api.git"},
		&entities.BuildSetting{
			ID:                   "setting-1",
			Name:                 "default",
			GitRef:               "main",
			DockerfilePath:       "apps/api/Dockerfile",
			BuildContext:         "apps/api",
			ImageName:            "demo/api",
			Platforms:            "linux/amd64",
			RegistryCacheEnabled: coreBoolPtr(true),
		},
		&entities.ContainerRegistry{
			Provider: entities.RegistryProviderGHCR,
			Endpoint: "ghcr.io",
		},
		&entities.Env{Base: entities.Base{ID: "env-1"}, Slug: "dev", ClusterNamespace: "project-dev", ClusterID: "cluster-1"},
		&entities.Project{Base: entities.Base{ID: "project-1"}, Slug: "demo"},
		"demo-api-default",
	)
	if err != nil {
		t.Fatalf("CreateBuildClientJobFromCodeRepo returned error: %v", err)
	}

	if job.Namespace != "project-dev" {
		t.Fatalf("expected job namespace project-dev, got %q", job.Namespace)
	}
	if job.Name != "build-demo-api-default-7" {
		t.Fatalf("unexpected job name %q", job.Name)
	}
}

func TestCreateBuildClientJob_UsesBuildctlContainerAndRemoteBuilderAddress(t *testing.T) {
	job, err := CreateBuildClientJobFromCodeRepo(
		&entities.Build{ID: "build-1", BuildNumber: 7, GitRef: "main", ImageFullName: "demo/api:v1.2.3"},
		&entities.CodeRepository{Name: "demo-api", GitRepoURL: "https://github.com/example/demo-api.git"},
		&entities.BuildSetting{
			ID:                   "setting-1",
			Name:                 "default",
			GitRef:               "main",
			DockerfilePath:       "apps/api/Dockerfile",
			BuildContext:         "apps/api",
			ImageName:            "demo/api",
			Platforms:            "linux/amd64,linux/arm64",
			RegistryCacheEnabled: coreBoolPtr(true),
			RegistryCacheRef:     "ghcr.io/demo/api:buildcache-setting-1",
		},
		&entities.ContainerRegistry{
			Provider: entities.RegistryProviderGHCR,
			Endpoint: "ghcr.io",
		},
		&entities.Env{Base: entities.Base{ID: "env-1"}, Slug: "dev", ClusterNamespace: "project-dev", ClusterID: "cluster-1"},
		&entities.Project{Base: entities.Base{ID: "project-1"}, Slug: "demo"},
		"demo-api-default",
	)
	if err != nil {
		t.Fatalf("CreateBuildClientJobFromCodeRepo returned error: %v", err)
	}

	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected one main container, got %#v", job.Spec.Template.Spec.Containers)
	}

	container := job.Spec.Template.Spec.Containers[0]
	if container.Name != "buildctl" {
		t.Fatalf("expected buildctl container, got %q", container.Name)
	}
	if len(container.Command) == 0 || container.Command[0] != "buildctl" {
		t.Fatalf("expected buildctl command, got %#v", container.Command)
	}
	if !containsArg(container.Args, "--addr=tcp://ketches-buildkitd.ketches-build.svc.cluster.local:1234") {
		t.Fatalf("expected remote builder address in args, got %#v", container.Args)
	}
	if !containsArg(container.Args, "--opt=platform=linux/amd64,linux/arm64") {
		t.Fatalf("expected multi-platform opt in args, got %#v", container.Args)
	}
}

func TestCreateBuildClientJob_MountsWorkspaceAndDockerConfig(t *testing.T) {
	job, err := CreateBuildClientJobFromCodeRepo(
		&entities.Build{ID: "build-1", BuildNumber: 7, GitRef: "main", ImageFullName: "demo/api:v1.2.3"},
		&entities.CodeRepository{Name: "demo-api", GitRepoURL: "https://github.com/example/demo-api.git"},
		&entities.BuildSetting{
			ID:                   "setting-1",
			Name:                 "default",
			GitRef:               "main",
			DockerfilePath:       "apps/api/Dockerfile",
			BuildContext:         "apps/api",
			ImageName:            "demo/api",
			Platforms:            "linux/amd64",
			RegistryCacheEnabled: coreBoolPtr(true),
		},
		&entities.ContainerRegistry{
			Provider: entities.RegistryProviderGHCR,
			Endpoint: "ghcr.io",
		},
		&entities.Env{Base: entities.Base{ID: "env-1"}, Slug: "dev", ClusterNamespace: "project-dev", ClusterID: "cluster-1"},
		&entities.Project{Base: entities.Base{ID: "project-1"}, Slug: "demo"},
		"demo-api-default",
	)
	if err != nil {
		t.Fatalf("CreateBuildClientJobFromCodeRepo returned error: %v", err)
	}

	if len(job.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("expected one init container, got %#v", job.Spec.Template.Spec.InitContainers)
	}
	if !hasMount(job.Spec.Template.Spec.InitContainers[0].VolumeMounts, "workspace", "/workspace") {
		t.Fatalf("expected workspace mount on init container, got %#v", job.Spec.Template.Spec.InitContainers[0].VolumeMounts)
	}
	if !hasMount(job.Spec.Template.Spec.Containers[0].VolumeMounts, "workspace", "/workspace") {
		t.Fatalf("expected workspace mount on buildctl container, got %#v", job.Spec.Template.Spec.Containers[0].VolumeMounts)
	}
	if !hasMount(job.Spec.Template.Spec.Containers[0].VolumeMounts, "docker-config", "/root/.docker") {
		t.Fatalf("expected docker config mount on buildctl container, got %#v", job.Spec.Template.Spec.Containers[0].VolumeMounts)
	}
}

func hasMount(mounts []corev1.VolumeMount, name, mountPath string) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == mountPath {
			return true
		}
	}
	return false
}

func coreBoolPtr(v bool) *bool {
	return &v
}
