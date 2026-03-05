package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KanikoImage   = "gcr.io/kaniko-project/executor:latest"
	GitCloneImage = "alpine/git:latest"
	BuildLabelKey = "app.ketches.cn/build"
	BuildAppLabel = "app.ketches.cn/app-slug"
	BuildIDLabel  = "app.ketches.cn/build-id"
)

// CreateBuildJob creates a Kubernetes Job to build a container image using Kaniko.
func CreateBuildJob(
	build *entities.Build,
	config *entities.AppBuildConfig,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	appSlug string,
) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("build-%s-%d", appSlug, build.BuildNumber)
	namespace := buildEnv.ClusterNamespace

	// Build the full image destination (build.ImageFullName may be short "image:tag" or already full)
	imageDestination := resolveImageDestination(registry, config.ImageName, build.ImageFullName)

	// Docker config for registry auth
	dockerConfigSecret := fmt.Sprintf("%s-build-registry", jobName)
	gitSecretName := fmt.Sprintf("%s-git-cred", jobName)

	backoffLimit := int32(0)
	ttl := int32(3600)

	kanikoArgs := buildKanikoArgs(config.DockerfilePath, config.BuildContext, imageDestination, config.BuildArgs, registry)

	// Git clone command
	gitRef := build.GitRef
	if gitRef == "" {
		gitRef = "main"
	}

	gitCloneCmd := buildGitCloneCommand(config.GitRepoURL, gitRef, config.GitUsername, config.GitPassword)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				BuildLabelKey: "true",
				BuildAppLabel: appSlug,
				BuildIDLabel:  build.ID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						BuildLabelKey: "true",
						BuildAppLabel: appSlug,
						BuildIDLabel:  build.ID,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:    "git-clone",
							Image:   GitCloneImage,
							Command: []string{"sh", "-c", gitCloneCmd},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workspace",
									MountPath: "/workspace",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "kaniko",
							Image: KanikoImage,
							Args:  kanikoArgs,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workspace",
									MountPath: "/workspace",
								},
								{
									Name:      "docker-config",
									MountPath: "/kaniko/.docker",
								},
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
							// SecurityContext: &corev1.SecurityContext{
							// 	Privileged: pointer.Bool(false),
							// 	RunAsUser:  pointer.Int64(0),
							// 	Capabilities: &corev1.Capabilities{
							// 		Drop: []corev1.Capability{"ALL"},
							// 	},
							// },
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "docker-config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: dockerConfigSecret,
								},
							},
						},
					},
				},
			},
		},
	}

	// If git credentials use a secret, mount it
	if config.GitUsername != "" && config.GitPassword != "" {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "git-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: gitSecretName,
				},
			},
		})
	}

	return job, nil
}

// CreateBuildJobFromCodeRepo creates a Kaniko Job using CodeRepository (git) + CodeRepositoryBuildConfig (dockerfile/image).
func CreateBuildJobFromCodeRepo(
	build *entities.Build,
	repo *entities.CodeRepository,
	config *entities.CodeRepositoryBuildConfig,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	jobSlug string,
) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("build-%s-%d", jobSlug, build.BuildNumber)
	namespace := buildEnv.ClusterNamespace
	imageDestination := resolveImageDestination(registry, config.ImageName, build.ImageFullName)
	dockerConfigSecret := fmt.Sprintf("%s-build-registry", jobName)
	gitSecretName := fmt.Sprintf("%s-git-cred", jobName)
	backoffLimit := int32(0)
	ttl := int32(3600)

	kanikoArgs := buildKanikoArgs(config.DockerfilePath, config.BuildContext, imageDestination, config.BuildArgs, registry)

	gitRef := build.GitRef
	if gitRef == "" {
		gitRef = config.GitRef
	}
	if gitRef == "" {
		gitRef = "main"
	}
	gitCloneCmd := buildGitCloneCommand(repo.GitRepoURL, gitRef, repo.GitUsername, repo.GitPassword)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				BuildLabelKey: "true",
				BuildAppLabel: jobSlug,
				BuildIDLabel:  build.ID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						BuildLabelKey: "true",
						BuildAppLabel: jobSlug,
						BuildIDLabel:  build.ID,
					},
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
							Name:  "kaniko",
							Image: KanikoImage,
							Args:  kanikoArgs,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "workspace", MountPath: "/workspace"},
								{Name: "docker-config", MountPath: "/kaniko/.docker"},
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
							// SecurityContext: &corev1.SecurityContext{
							// 	RunAsUser:                pointer.Int64(0),
							// 	AllowPrivilegeEscalation: pointer.Bool(false),
							// },
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

// CreateBuildSecretsFromCodeRepo creates K8s secrets for a code-repo build job.
func CreateBuildSecretsFromCodeRepo(
	ctx context.Context,
	build *entities.Build,
	repo *entities.CodeRepository,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	jobSlug string,
) error {
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster client: %w", err)
	}
	jobName := fmt.Sprintf("build-%s-%d", jobSlug, build.BuildNumber)
	namespace := buildEnv.ClusterNamespace

	dockerConfigJSON := buildDockerConfigJSON(registry)
	registrySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-build-registry", jobName),
			Namespace: namespace,
			Labels:    map[string]string{BuildLabelKey: "true", BuildIDLabel: build.ID},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"config.json": dockerConfigJSON},
	}
	if _, err := client.CoreV1().Secrets(namespace).Create(ctx, registrySecret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create registry secret: %w", err)
	}
	if repo.GitUsername != "" && repo.GitPassword != "" {
		gitSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-git-cred", jobName),
				Namespace: namespace,
				Labels:    map[string]string{BuildLabelKey: "true", BuildIDLabel: build.ID},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: map[string]string{"username": repo.GitUsername, "password": repo.GitPassword},
		}
		if _, err := client.CoreV1().Secrets(namespace).Create(ctx, gitSecret, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create git secret: %w", err)
		}
	}
	return nil
}

// SubmitBuildJobFromCodeRepo creates secrets and submits the Kaniko job for a code repository build.
func SubmitBuildJobFromCodeRepo(
	ctx context.Context,
	build *entities.Build,
	repo *entities.CodeRepository,
	config *entities.CodeRepositoryBuildConfig,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	jobSlug string,
) (string, string, error) {
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get cluster client: %w", err)
	}
	if err := CreateBuildSecretsFromCodeRepo(ctx, build, repo, registry, buildEnv, jobSlug); err != nil {
		return "", "", err
	}
	job, err := CreateBuildJobFromCodeRepo(build, repo, config, registry, buildEnv, jobSlug)
	if err != nil {
		return "", "", err
	}
	createdJob, err := client.BatchV1().Jobs(buildEnv.ClusterNamespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to create build job: %w", err)
	}
	return createdJob.Name, createdJob.Namespace, nil
}

// CreateBuildSecrets creates the Kubernetes secrets needed for a build job.
func CreateBuildSecrets(
	ctx context.Context,
	build *entities.Build,
	config *entities.AppBuildConfig,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	appSlug string,
) error {
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster client: %w", err)
	}

	jobName := fmt.Sprintf("build-%s-%d", appSlug, build.BuildNumber)
	namespace := buildEnv.ClusterNamespace

	// Create docker config secret for registry auth
	dockerConfigJSON := buildDockerConfigJSON(registry)
	registrySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-build-registry", jobName),
			Namespace: namespace,
			Labels: map[string]string{
				BuildLabelKey: "true",
				BuildIDLabel:  build.ID,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"config.json": dockerConfigJSON,
		},
	}

	if _, err := client.CoreV1().Secrets(namespace).Create(ctx, registrySecret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create registry secret: %w", err)
	}

	// Create git credentials secret if needed
	if config.GitUsername != "" && config.GitPassword != "" {
		gitSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-git-cred", jobName),
				Namespace: namespace,
				Labels: map[string]string{
					BuildLabelKey: "true",
					BuildIDLabel:  build.ID,
				},
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"username": config.GitUsername,
				"password": config.GitPassword,
			},
		}

		if _, err := client.CoreV1().Secrets(namespace).Create(ctx, gitSecret, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create git secret: %w", err)
		}
	}

	return nil
}

// SubmitBuildJob creates and submits the Kaniko build job to the cluster.
func SubmitBuildJob(
	ctx context.Context,
	build *entities.Build,
	config *entities.AppBuildConfig,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	appSlug string,
) (string, string, error) {
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get cluster client: %w", err)
	}

	// Create secrets first
	if err := CreateBuildSecrets(ctx, build, config, registry, buildEnv, appSlug); err != nil {
		return "", "", err
	}

	// Create the job
	job, err := CreateBuildJob(build, config, registry, buildEnv, appSlug)
	if err != nil {
		return "", "", err
	}

	createdJob, err := client.BatchV1().Jobs(buildEnv.ClusterNamespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to create build job: %w", err)
	}

	return createdJob.Name, createdJob.Namespace, nil
}

// CancelBuildJob deletes the build job in the cluster.
func CancelBuildJob(ctx context.Context, clusterID, jobName, jobNamespace string) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	propagation := metav1.DeletePropagationBackground
	return client.BatchV1().Jobs(jobNamespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
}

// CleanupBuildSecrets removes the secrets created for a build job.
func CleanupBuildSecrets(ctx context.Context, clusterID, buildID, namespace string) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return
	}

	secrets, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", BuildIDLabel, buildID),
	})
	if err != nil {
		return
	}

	for _, s := range secrets.Items {
		_ = client.CoreV1().Secrets(namespace).Delete(ctx, s.Name, metav1.DeleteOptions{})
	}
}

// resolveImageDestination returns the full image reference for Kaniko --destination.
// imageNameOrFull may be "image:tag" (short) or already a full reference; we use registry to build full if needed.
func resolveImageDestination(registry *entities.ContainerRegistry, imageName, imageNameOrFull string) string {
	imageNameOrFull = sanitizeImageReference(imageNameOrFull)

	tag := imageNameOrFull
	if idx := strings.LastIndex(imageNameOrFull, ":"); idx >= 0 && idx < len(imageNameOrFull)-1 {
		tag = imageNameOrFull[idx+1:]
		imageName = imageNameOrFull[:idx]
		if strings.Contains(imageName, "/") {
			// Already full reference
			return imageNameOrFull
		}
	}
	return BuildImageFullName(registry, imageName, tag)
}

// BuildImageFullName returns the full image reference (registry/namespace/image:tag).
// For Docker Hub, the URL is omitted (namespace/image:tag or image:tag) so pull uses default docker.io.
func BuildImageFullName(registry *entities.ContainerRegistry, imageName, tag string) string {
	imageWithTag := imageName + ":" + tag

	if registry.Provider == entities.RegistryProviderDockerHub {
		if registry.Namespace != "" {
			return registry.Namespace + "/" + imageWithTag
		}
		return imageWithTag
	}

	endpoint := normalizeRegistryHost(registry.Endpoint)
	if registry.Namespace != "" {
		return fmt.Sprintf("%s/%s/%s", endpoint, registry.Namespace, imageWithTag)
	}
	return fmt.Sprintf("%s/%s", endpoint, imageWithTag)
}

func sanitizeImageReference(imageRef string) string {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return imageRef
	}
	if after, ok := strings.CutPrefix(imageRef, "https://"); ok {
		return after
	}
	if after, ok := strings.CutPrefix(imageRef, "http://"); ok {
		return after
	}
	return imageRef
}

func normalizeRegistryHost(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimSuffix(endpoint, "/")
	if after, ok := strings.CutPrefix(endpoint, "https://"); ok {
		endpoint = after
	}
	if after, ok := strings.CutPrefix(endpoint, "http://"); ok {
		endpoint = after
	}
	endpoint = strings.TrimPrefix(endpoint, "//")
	return strings.TrimSuffix(endpoint, "/")
}

func buildKanikoArgs(dockerfilePath, buildContext, imageDestination, buildArgsJSON string, registry *entities.ContainerRegistry) []string {
	kanikoArgs := []string{
		fmt.Sprintf("--dockerfile=%s", dockerfilePath),
		fmt.Sprintf("--context=dir:///workspace/%s", buildContext),
		fmt.Sprintf("--destination=%s", imageDestination),
		"--snapshot-mode=redo",
		// Avoid failing the whole build when Kaniko cannot cleanup transient files on some runtimes.
		"--cleanup=false",
	}

	if shouldEnableKanikoCache(registry) {
		kanikoArgs = append(kanikoArgs, "--cache=true")
	} else {
		// Docker Hub cache pushes commonly fail due to separate cache repo auth scope.
		kanikoArgs = append(kanikoArgs, "--cache=false")
	}

	buildArgs := parseBuildArgs(buildArgsJSON)
	buildArgs = withDefaultPlatformBuildArgs(buildArgs)
	for _, arg := range buildArgs {
		kanikoArgs = append(kanikoArgs, fmt.Sprintf("--build-arg=%s", arg))
	}

	return kanikoArgs
}

func shouldEnableKanikoCache(registry *entities.ContainerRegistry) bool {
	if registry == nil {
		return false
	}
	return registry.Provider != entities.RegistryProviderDockerHub
}

func parseBuildArgs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var args []string

	// Preferred format: JSON object, e.g. {"KEY":"VALUE"}
	var jsonArgs map[string]string
	if err := json.Unmarshal([]byte(raw), &jsonArgs); err == nil && len(jsonArgs) > 0 {
		for k, v := range jsonArgs {
			args = append(args, fmt.Sprintf("%s=%s", strings.TrimSpace(k), v))
		}
		return args
	}

	// Compatibility format: KEY1=val1,KEY2=val2 or newline separated pairs.
	normalized := strings.ReplaceAll(raw, "\n", ",")
	for _, part := range strings.Split(normalized, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "=") {
			key, val, _ := strings.Cut(part, "=")
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			if key == "" {
				continue
			}
			args = append(args, fmt.Sprintf("%s=%s", key, val))
			continue
		}

		// Allow KEY form so Kaniko can resolve from environment if present.
		args = append(args, part)
	}

	return args
}

func withDefaultPlatformBuildArgs(args []string) []string {
	existing := make(map[string]bool)
	for _, arg := range args {
		key := arg
		if k, _, ok := strings.Cut(arg, "="); ok {
			key = strings.TrimSpace(k)
		}
		if key == "" {
			continue
		}
		existing[key] = true
	}

	arch := runtime.GOARCH
	os := runtime.GOOS
	platform := fmt.Sprintf("%s/%s", os, arch)

	defaults := []string{
		fmt.Sprintf("BUILDPLATFORM=%s", platform),
		fmt.Sprintf("TARGETPLATFORM=%s", platform),
		fmt.Sprintf("TARGETOS=%s", os),
		fmt.Sprintf("TARGETARCH=%s", arch),
	}

	for _, kv := range defaults {
		k, _, _ := strings.Cut(kv, "=")
		if existing[k] {
			continue
		}
		args = append(args, kv)
	}

	return args
}

func buildDockerConfigJSON(registry *entities.ContainerRegistry) []byte {
	// endpoint := registry.Endpoint
	// if registry.Provider == entities.RegistryProviderDockerHub {
	// 	endpoint = "https://index.docker.io/v1/"
	// }

	// auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", registry.Username, registry.Password)))

	// config := map[string]any{
	// 	"auths": map[string]any{
	// 		endpoint: map[string]string{
	// 			"auth": auth,
	// 		},
	// 	},
	// }

	// data, _ := json.Marshal(config)
	// return data

	var auths map[string]any

	if registry.Provider == entities.RegistryProviderDockerHub {
		// For Docker Hub, configure auth for both v1 and v2 endpoints
		auth := base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", registry.Username, registry.Password))
		auths = map[string]any{
			"https://index.docker.io/v1/": map[string]string{
				"auth": auth,
			},
			"https://index.docker.io/v2/": map[string]string{
				"auth": auth,
			},
			"index.docker.io": map[string]string{
				"auth": auth,
			},
			"registry-1.docker.io": map[string]string{
				"auth": auth,
			},
		}
	} else {
		endpoint := normalizeRegistryHost(registry.Endpoint)
		auth := base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", registry.Username, registry.Password))
		auths = map[string]any{
			endpoint: map[string]string{
				"auth": auth,
			},
			"https://" + endpoint: map[string]string{
				"auth": auth,
			},
		}
	}

	config := map[string]any{
		"auths": auths,
	}

	data, _ := json.Marshal(config)
	return data
}

func buildGitCloneCommand(repoURL, ref, username, password string) string {
	cloneURL := convertSSHToHTTPS(repoURL)
	if username != "" && password != "" {
		cloneURL = injectGitCredentials(cloneURL, username, password)
	}

	// Clone the repo, checkout the specified ref
	return fmt.Sprintf(
		"git clone --depth 1 --branch %s %s /workspace || (git clone %s /workspace && cd /workspace && git checkout %s)",
		ref, cloneURL, cloneURL, ref,
	)
}

func convertSSHToHTTPS(repoURL string) string {
	if after, ok := strings.CutPrefix(repoURL, "git@"); ok {
		repoURL = after
		repoURL = strings.Replace(repoURL, ":", "/", 1)
		return "https://" + repoURL
	}
	if after, ok := strings.CutPrefix(repoURL, "ssh://git@"); ok {
		return "https://" + after
	}
	return repoURL
}

func injectGitCredentials(repoURL, username, password string) string {
	if after, ok := strings.CutPrefix(repoURL, "https://"); ok {
		return fmt.Sprintf("https://%s:%s@%s", username, password, after)
	}
	if after, ok := strings.CutPrefix(repoURL, "http://"); ok {
		return fmt.Sprintf("http://%s:%s@%s", username, password, after)
	}
	return repoURL
}
