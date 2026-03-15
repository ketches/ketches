package core

import (
	"fmt"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateBuildClientJob(
	build *entities.Build,
	repo *entities.CodeRepository,
	setting *entities.BuildSetting,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	appCtx *models.AppContext,
) (*batchv1.Job, error) {
	appSlug := appCtx.App.Slug
	gitRef := build.GitRef
	if gitRef == "" {
		gitRef = "main"
	}
	gitCloneCmd := buildGitCloneCommand(repo.GitRepoURL, gitRef, repo.GitUsername, repo.GitPassword)

	labels := map[string]string{
		kube.LabelAppID:       appCtx.App.ID,
		kube.LabelEnvID:       appCtx.EnvContext.Env.ID,
		kube.LabelEnvSlug:     appCtx.EnvContext.Env.Slug,
		kube.LabelProjectID:   appCtx.EnvContext.Project.ID,
		kube.LabelProjectSlug: appCtx.EnvContext.Project.Slug,
		kube.LabelBuildKey:    "true",
		kube.LabelAppSlug:     appSlug,
		kube.LabelBuildID:     build.ID,
		kube.LabelManagedBy:   "true",
	}

	return createBuildClientJob(
		fmt.Sprintf("build-%s-%d", appSlug, build.BuildNumber),
		build,
		repo,
		setting,
		registry,
		buildEnv.ClusterNamespace,
		gitCloneCmd,
		labels,
	)
}

func CreateBuildClientJobFromCodeRepo(
	build *entities.Build,
	repo *entities.CodeRepository,
	setting *entities.BuildSetting,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	project *entities.Project,
	jobSlug string,
) (*batchv1.Job, error) {
	gitRef := build.GitRef
	if gitRef == "" {
		gitRef = setting.GitRef
	}
	if gitRef == "" {
		gitRef = "main"
	}
	gitCloneCmd := buildGitCloneCommand(repo.GitRepoURL, gitRef, repo.GitUsername, repo.GitPassword)

	labels := map[string]string{
		kube.LabelEnvID:       buildEnv.ID,
		kube.LabelEnvSlug:     buildEnv.Slug,
		kube.LabelProjectID:   project.ID,
		kube.LabelProjectSlug: project.Slug,
		kube.LabelBuildKey:    "true",
		kube.LabelAppSlug:     jobSlug,
		kube.LabelBuildID:     build.ID,
		kube.LabelManagedBy:   "true",
	}

	return createBuildClientJob(
		fmt.Sprintf("build-%s-%d", jobSlug, build.BuildNumber),
		build,
		repo,
		setting,
		registry,
		buildEnv.ClusterNamespace,
		gitCloneCmd,
		labels,
	)
}

func createBuildClientJob(
	jobName string,
	build *entities.Build,
	repo *entities.CodeRepository,
	setting *entities.BuildSetting,
	registry *entities.ContainerRegistry,
	namespace string,
	gitCloneCmd string,
	labels map[string]string,
) (*batchv1.Job, error) {
	imageDestination := resolveImageDestination(registry, setting.ImageName, build.ImageFullName)
	dockerConfigSecret := fmt.Sprintf("%s-build-registry", jobName)
	gitSecretName := fmt.Sprintf("%s-git-cred", jobName)
	backoffLimit := int32(0)
	ttl := int32(3600)

	cacheRef := setting.RegistryCacheRef
	if buildSettingRegistryCacheEnabled(setting.RegistryCacheEnabled) && cacheRef == "" {
		cacheRef = defaultRegistryCacheRef(imageDestination, setting.ID)
	}

	buildctlArgv, err := buildctlArgs(BuildkitOptions{
		DockerfilePath:       setting.DockerfilePath,
		BuildContext:         setting.BuildContext,
		ImageDestination:     imageDestination,
		Platforms:            setting.Platforms,
		RegistryCacheEnabled: buildSettingRegistryCacheEnabled(setting.RegistryCacheEnabled),
		RegistryCacheRef:     cacheRef,
		BuildArgs:            parseBuildArgs(setting.BuildArgs),
	})
	if err != nil {
		return nil, err
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:    "git-clone",
							Image:   GitCloneImage,
							Command: []string{"sh", "-c", gitCloneCmd},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "workspace", MountPath: "/workspace"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "buildctl",
							Image:   buildctlImage,
							Command: []string{"buildctl"},
							Args:    buildctlArgv,
							Env: []corev1.EnvVar{
								{Name: "DOCKER_CONFIG", Value: "/root/.docker"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "workspace", MountPath: "/workspace"},
								{Name: "docker-config", MountPath: "/root/.docker"},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("4Gi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "workspace", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "docker-config", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: dockerConfigSecret}}},
					},
				},
			},
		},
	}

	if repo.GitUsername != "" && repo.GitPassword != "" {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "git-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: gitSecretName},
			},
		})
	}

	return job, nil
}

func buildSettingRegistryCacheEnabled(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}
