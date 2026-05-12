package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KanikoImage   = "gcr.io/kaniko-project/executor:latest"
	GitCloneImage = "alpine/git:latest"
)

// CreateBuildJob creates a Kubernetes Job to build a container image via a BuildKit client job.
func CreateBuildJob(
	build *entities.Build,
	repo *entities.CodeRepository,
	setting *entities.BuildSetting,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	appCtx *models.AppContext,
) (*batchv1.Job, error) {
	return CreateBuildClientJob(build, repo, setting, registry, buildEnv, appCtx)
}

// CreateBuildJobFromCodeRepo creates a BuildKit client Job using CodeRepository (git) + BuildSetting (dockerfile/image).
func CreateBuildJobFromCodeRepo(
	build *entities.Build,
	repo *entities.CodeRepository,
	setting *entities.BuildSetting,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	project *entities.Project,
	jobSlug string,
) (*batchv1.Job, error) {
	return CreateBuildClientJobFromCodeRepo(build, repo, setting, registry, buildEnv, project, jobSlug)
}

// CreateBuildSecretsFromCodeRepo creates K8s secrets for a code-repo build job.
func CreateBuildSecretsFromCodeRepo(
	ctx context.Context,
	build *entities.Build,
	repo *entities.CodeRepository,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	project *entities.Project,
	jobSlug string,
) error {
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		return app.WrapError("failed to get cluster client", err)
	}

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

	jobName := fmt.Sprintf("build-%s-%d", jobSlug, build.BuildNumber)
	namespace := buildEnv.ClusterNamespace

	dockerConfigJSON, err := buildDockerConfigJSON(registry)
	if err != nil {
		return err
	}
	registrySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-build-registry", jobName),
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"config.json": dockerConfigJSON},
	}
	if _, err := client.CoreV1().Secrets(namespace).Create(ctx, registrySecret, metav1.CreateOptions{}); err != nil {
		return app.WrapError("failed to create registry secret", err)
	}
	plaintextGitPassword, err := resolveCodeRepositoryGitPassword(repo)
	if err != nil {
		return err
	}
	if repo.GitUsername != "" && plaintextGitPassword != "" {
		gitSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-git-cred", jobName),
				Namespace: namespace,
				Labels:    labels,
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: map[string]string{"username": repo.GitUsername, "password": plaintextGitPassword},
		}
		if _, err := client.CoreV1().Secrets(namespace).Create(ctx, gitSecret, metav1.CreateOptions{}); err != nil {
			return app.WrapError("failed to create git secret", err)
		}
	}
	return nil
}

// SubmitBuildJobFromCodeRepo creates secrets and submits the BuildKit client job for a code repository build.
func SubmitBuildJobFromCodeRepo(
	ctx context.Context,
	build *entities.Build,
	repo *entities.CodeRepository,
	setting *entities.BuildSetting,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	project *entities.Project,
	jobSlug string,
) (string, string, error) {
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		return "", "", app.WrapError("failed to get cluster client", err)
	}
	if err := EnsureClusterBuildkitInfrastructure(ctx, buildEnv.ClusterID); err != nil {
		return "", "", app.WrapError("failed to ensure buildkit infrastructure", err)
	}
	if err := EnsureBuildkitBuildPrerequisites(ctx, buildEnv.ClusterID, setting.Platforms); err != nil {
		return "", "", app.WrapError("failed to validate buildkit readiness", err)
	}
	if err := CreateBuildSecretsFromCodeRepo(ctx, build, repo, registry, buildEnv, project, jobSlug); err != nil {
		return "", "", err
	}
	job, err := CreateBuildClientJobFromCodeRepo(build, repo, setting, registry, buildEnv, project, jobSlug)
	if err != nil {
		return "", "", err
	}
	createdJob, err := client.BatchV1().Jobs(buildEnv.ClusterNamespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", "", app.WrapError("failed to create build job", err)
	}
	return createdJob.Name, createdJob.Namespace, nil
}

// CreateBuildSecrets creates the Kubernetes secrets needed for a build job.
func CreateBuildSecrets(
	ctx context.Context,
	build *entities.Build,
	repo *entities.CodeRepository,
	setting *entities.BuildSetting,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	appCtx *models.AppContext,
) error {
	appSlug := appCtx.App.Slug
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		return app.WrapError("failed to get cluster client", err)
	}

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

	jobName := fmt.Sprintf("build-%s-%d", appSlug, build.BuildNumber)
	namespace := buildEnv.ClusterNamespace

	// Create docker config secret for registry auth
	dockerConfigJSON, err := buildDockerConfigJSON(registry)
	if err != nil {
		return err
	}
	registrySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-build-registry", jobName),
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"config.json": dockerConfigJSON,
		},
	}

	if _, err := client.CoreV1().Secrets(namespace).Create(ctx, registrySecret, metav1.CreateOptions{}); err != nil {
		return app.WrapError("failed to create registry secret", err)
	}

	plaintextGitPassword, err := resolveCodeRepositoryGitPassword(repo)
	if err != nil {
		return err
	}

	// Create git credentials secret if needed
	if repo.GitUsername != "" && plaintextGitPassword != "" {
		gitSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-git-cred", jobName),
				Namespace: namespace,
				Labels:    labels,
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"username": repo.GitUsername,
				"password": plaintextGitPassword,
			},
		}

		if _, err := client.CoreV1().Secrets(namespace).Create(ctx, gitSecret, metav1.CreateOptions{}); err != nil {
			return app.WrapError("failed to create git secret", err)
		}
	}

	return nil
}

// SubmitBuildJob creates and submits the BuildKit client job to the cluster.
func SubmitBuildJob(
	ctx context.Context,
	build *entities.Build,
	repo *entities.CodeRepository,
	setting *entities.BuildSetting,
	registry *entities.ContainerRegistry,
	buildEnv *entities.Env,
	appCtx *models.AppContext,
) (string, string, error) {
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		return "", "", app.WrapError("failed to get cluster client", err)
	}
	if err := EnsureClusterBuildkitInfrastructure(ctx, buildEnv.ClusterID); err != nil {
		return "", "", app.WrapError("failed to ensure buildkit infrastructure", err)
	}
	if err := EnsureBuildkitBuildPrerequisites(ctx, buildEnv.ClusterID, setting.Platforms); err != nil {
		return "", "", app.WrapError("failed to validate buildkit readiness", err)
	}

	// Create secrets first
	if err := CreateBuildSecrets(ctx, build, repo, setting, registry, buildEnv, appCtx); err != nil {
		return "", "", err
	}

	// Create the job
	job, err := CreateBuildClientJob(build, repo, setting, registry, buildEnv, appCtx)
	if err != nil {
		return "", "", err
	}

	createdJob, err := client.BatchV1().Jobs(buildEnv.ClusterNamespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", "", app.WrapError("failed to create build job", err)
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
		LabelSelector: fmt.Sprintf("%s=%s", kube.LabelBuildID, buildID),
	})
	if err != nil {
		return
	}

	for _, s := range secrets.Items {
		_ = client.CoreV1().Secrets(namespace).Delete(ctx, s.Name, metav1.DeleteOptions{})
	}
}

// resolveImageDestination returns the full image reference for the pushed build output.
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

	if registry.Namespace != "" {
		return fmt.Sprintf("%s/%s/%s", registry.Endpoint, registry.Namespace, imageWithTag)
	}
	return fmt.Sprintf("%s/%s", registry.Endpoint, imageWithTag)
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

func parseBuildArgs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var args []string
	for _, part := range strings.Split(raw, "\n") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		args = append(args, part)
	}

	return args
}

func buildDockerConfigJSON(registry *entities.ContainerRegistry) ([]byte, error) {
	endpoint := registry.Endpoint
	if registry.Provider == entities.RegistryProviderDockerHub {
		endpoint = "https://index.docker.io/v1/"
	}

	plaintextPassword, err := secrets.DecryptString(registry.Password)
	if err != nil {
		return nil, app.WrapError("decrypt registry password", err)
	}

	auth := base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", registry.Username, plaintextPassword))

	config := map[string]any{
		"auths": map[string]any{
			endpoint: map[string]string{
				"auth": auth,
			},
		},
	}

	data, _ := json.Marshal(config)
	return data, nil
}

func buildGitCloneCommand(repoURL, ref, username, password string) ([]string, []string) {
	cloneURL := convertSSHToHTTPS(repoURL)
	if username != "" && password != "" {
		cloneURL = injectGitCredentials(cloneURL, username, password)
	}

	script := `git clone --depth 1 --branch "$1" "$2" /workspace || (git clone "$2" /workspace && cd /workspace && git checkout "$1")`
	return []string{"sh", "-eu", "-c", script}, []string{"ketches-git-clone", ref, cloneURL}
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
