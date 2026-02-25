package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ListBuilds(appID string, page, pageSize int) (int64, []entities.Build, error) {
	var total int64
	var builds []entities.Build
	query := db.DB.Model(&entities.Build{}).Where("app_id = ?", appID)
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Order("build_number DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&builds).Error; err != nil {
		return 0, nil, err
	}
	return total, builds, nil
}

func GetBuild(buildID string) (*entities.Build, error) {
	var build entities.Build
	if err := db.DB.Preload("BuildEnv.Cluster").
		First(&build, "id = ?", buildID).Error; err != nil {
		return nil, err
	}
	return &build, nil
}

func TriggerBuild(appID, userID string, req *models.TriggerBuildRequest) (*entities.Build, error) {
	// Get app with env/project info
	app, err := GetApp(appID)
	if err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}

	// Get build config
	config, err := GetBuildConfig(appID)
	if err != nil {
		return nil, fmt.Errorf("build config not found: please configure build settings first")
	}

	// Get the registry
	registry, err := GetContainerRegistry(config.RegistryID)
	if err != nil {
		return nil, fmt.Errorf("image registry not found: %w", err)
	}

	// Get the build environment for this project
	buildEnv, err := GetProjectBuildEnv(app.Env.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("no build environment configured for this project: please set a build environment first")
	}

	// Check for active builds
	var activeCount int64
	if err := db.DB.Model(&entities.Build{}).
		Where("app_id = ? AND status IN ?", appID, []entities.BuildStatus{
			entities.BuildStatusPending,
			entities.BuildStatusCloning,
			entities.BuildStatusBuilding,
		}).Count(&activeCount).Error; err != nil {
		return nil, err
	}
	if activeCount > 0 {
		return nil, errors.New("an active build already exists for this app")
	}

	// Get next build number
	var lastBuild entities.Build
	var buildNumber int
	if err := db.DB.Where("app_id = ?", appID).Order("build_number DESC").First(&lastBuild).Error; err != nil {
		buildNumber = 1
	} else {
		buildNumber = lastBuild.BuildNumber + 1
	}

	// Determine git ref and image tag
	gitRef := config.GitRef
	if req.GitRef != "" {
		gitRef = req.GitRef
	}

	imageTag := fmt.Sprintf("%s-%d", sanitizeRef(gitRef), buildNumber)
	if req.ImageTag != "" {
		imageTag = req.ImageTag
	}

	imageFullName := fmt.Sprintf("%s:%s", config.ImageName, imageTag)

	// Determine auto deploy
	autoDeploy := config.AutoDeploy
	if req.AutoDeploy != nil {
		autoDeploy = *req.AutoDeploy
	}

	// Temporarily store autoDeploy in config for the watcher
	if autoDeploy != config.AutoDeploy {
		// We use the build-level config snapshot
	}

	now := time.Now()
	appIDPtr := appID
	buildConfigID := config.ID
	build := &entities.Build{
		Base:          entities.Base{ID: uuid.New()},
		AppID:         &appIDPtr,
		BuildConfigID: &buildConfigID,
		BuildNumber:   buildNumber,
		Status:        entities.BuildStatusPending,
		BuildEnvID:    buildEnv.ID,
		GitRepoURL:    config.GitRepoURL,
		GitRef:        gitRef,
		ImageFullName: imageFullName,
		TriggerType:   entities.BuildTriggerManual,
		TriggeredBy:   userID,
		StartedAt:     &now,
	}

	if err := db.DB.Create(build).Error; err != nil {
		return nil, err
	}

	// Submit the build job to Kubernetes
	jobName, jobNamespace, err := core.SubmitBuildJob(
		context.Background(),
		build,
		config,
		registry,
		buildEnv,
		app.Slug,
	)
	if err != nil {
		// Update build as failed
		build.Status = entities.BuildStatusFailed
		build.ErrorMessage = err.Error()
		completedAt := time.Now()
		build.CompletedAt = &completedAt
		db.DB.Save(build)
		return nil, fmt.Errorf("failed to submit build job: %w", err)
	}

	// Update build with job info
	build.JobName = jobName
	build.JobNamespace = jobNamespace
	if err := db.DB.Save(build).Error; err != nil {
		return nil, err
	}

	// Start watching the build
	core.GlobalBuildWatcher.StartWatching(build)

	return build, nil
}

func CancelBuild(buildID string) (*entities.Build, error) {
	build, err := GetBuild(buildID)
	if err != nil {
		return nil, err
	}

	if build.Status != entities.BuildStatusPending &&
		build.Status != entities.BuildStatusCloning &&
		build.Status != entities.BuildStatusBuilding {
		return nil, errors.New("build is not active")
	}

	// Stop watching
	core.GlobalBuildWatcher.StopWatching(buildID)

	// Cancel the K8s job
	if build.JobName != "" {
		if err := core.CancelBuildJob(
			context.Background(),
			build.BuildEnv.ClusterID,
			build.JobName,
			build.JobNamespace,
		); err != nil {
			log.Printf("Failed to cancel build job: %v", err)
		}
	}

	// Cleanup secrets
	go core.CleanupBuildSecrets(context.Background(), build.BuildEnv.ClusterID, buildID, build.JobNamespace)

	// Update build status
	now := time.Now()
	build.Status = entities.BuildStatusCancelled
	build.CompletedAt = &now
	if build.StartedAt != nil {
		build.Duration = int(now.Sub(*build.StartedAt).Seconds())
	}

	if err := db.DB.Save(build).Error; err != nil {
		return nil, err
	}

	return build, nil
}

func DeployBuild(buildID string) (*entities.Build, error) {
	build, err := GetBuild(buildID)
	if err != nil {
		return nil, err
	}

	if build.Status != entities.BuildStatusSucceeded {
		return nil, errors.New("can only deploy a successful build")
	}

	if build.ImageFullName == "" {
		return nil, errors.New("build has no image")
	}

	// Get build config for registry credentials (app-bound builds only)
	if build.BuildConfigID == nil || *build.BuildConfigID == "" {
		return nil, errors.New("build has no app build config; use code repository deploy for this build")
	}
	var config entities.AppBuildConfig
	if err := db.DB.Preload("Registry").First(&config, "id = ?", *build.BuildConfigID).Error; err != nil {
		return nil, err
	}

	// Get the app (app-bound builds only)
	if build.AppID == nil || *build.AppID == "" {
		return nil, errors.New("build has no associated app; use code repository deploy for this build")
	}
	var app entities.App
	if err := db.DB.Preload("Env.Cluster").
		Preload("EnvVars").
		Preload("Volumes").
		Preload("ConfigFiles").
		Preload("Probes").
		Preload("Gateways").
		Preload("AppPlugins.Plugin").
		Preload("AutoScaling").
		Preload("SchedulingRule").
		First(&app, "id = ?", *build.AppID).Error; err != nil {
		return nil, err
	}

	// Update app's container image
	app.ContainerImage = build.ImageFullName
	app.RegistryUsername = config.Registry.Username
	app.RegistryPassword = config.Registry.Password

	if err := db.DB.Save(&app).Error; err != nil {
		return nil, err
	}

	// Apply the app
	if err := core.ApplyApp(context.Background(), &app); err != nil {
		return nil, fmt.Errorf("failed to deploy: %w", err)
	}

	// Update deploy status
	app.DeployStatus = "deployed"
	db.DB.Model(&app).Update("deploy_status", "deployed")

	return build, nil
}

func RebuildBuild(buildID, userID string, req *models.RebuildRequest) (*entities.Build, error) {
	originalBuild, err := GetBuild(buildID)
	if err != nil {
		return nil, err
	}
	if originalBuild.AppID == nil || *originalBuild.AppID == "" {
		return nil, errors.New("this build is not associated with an app; trigger a new build from the code repository instead")
	}

	triggerReq := &models.TriggerBuildRequest{
		GitRef:   originalBuild.GitRef,
		ImageTag: req.ImageTag,
	}

	return TriggerBuild(*originalBuild.AppID, userID, triggerReq)
}

func StreamBuildLogs(c *gin.Context, buildID string) {
	build, err := GetBuild(buildID)
	if err != nil {
		c.JSON(404, gin.H{"error": "build not found"})
		return
	}

	if build.JobName == "" {
		c.JSON(400, gin.H{"error": "build has no job"})
		return
	}

	client, err := kube.GlobalClusterStore.GetClient(build.BuildEnv.ClusterID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get cluster client"})
		return
	}

	pods, err := client.CoreV1().Pods(build.JobNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", build.JobName),
	})
	if err != nil || len(pods.Items) == 0 {
		c.JSON(404, gin.H{"error": "build pod not found"})
		return
	}

	pod := &pods.Items[0]
	podName := pod.Name

	// Collect container names: init containers first, then main containers (so user sees init + build logs)
	var containerNames []string
	for _, ic := range pod.Spec.InitContainers {
		containerNames = append(containerNames, ic.Name)
	}
	for _, c := range pod.Spec.Containers {
		containerNames = append(containerNames, c.Name)
	}

	follow := build.Status == entities.BuildStatusPending ||
		build.Status == entities.BuildStatusCloning ||
		build.Status == entities.BuildStatusBuilding

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	tailLines := int64(1000)
	buf := make([]byte, 4096)

	for i, containerName := range containerNames {
		// Init containers: do not follow (read until EOF). Last container (kaniko): follow if build still running.
		containerFollow := follow && (i == len(containerNames)-1)
		logReq := client.CoreV1().Pods(build.JobNamespace).GetLogs(podName, &corev1.PodLogOptions{
			Container:  containerName,
			Follow:     containerFollow,
			TailLines:  &tailLines,
			Timestamps: true,
		})

		stream, err := logReq.Stream(context.Background())
		if err != nil {
			// Container may not have started yet; send a line and continue
			c.SSEvent("log", fmt.Sprintf("[%s] (logs not available yet)\n", containerName))
			c.Writer.Flush()
			continue
		}
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				c.SSEvent("log", string(buf[:n]))
				c.Writer.Flush()
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("Build log stream error %s: %v", containerName, err)
				}
				stream.Close()
				break
			}
		}
	}

	c.SSEvent("done", "stream ended")
	c.Writer.Flush()
}

func ToBuildResponse(b *entities.Build) models.BuildResponse {
	codeRepoID := ""
	if b.CodeRepositoryID != nil {
		codeRepoID = *b.CodeRepositoryID
	}
	codeRepoConfigID := ""
	if b.CodeRepositoryBuildConfigID != nil {
		codeRepoConfigID = *b.CodeRepositoryBuildConfigID
	}
	appID := ""
	if b.AppID != nil {
		appID = *b.AppID
	}
	buildConfigID := ""
	if b.BuildConfigID != nil {
		buildConfigID = *b.BuildConfigID
	}

	resp := models.BuildResponse{
		ID:                          b.ID,
		CodeRepositoryID:            codeRepoID,
		CodeRepositoryBuildConfigID: codeRepoConfigID,
		AppID:                       appID,
		BuildConfigID:               buildConfigID,
		BuildNumber:                 b.BuildNumber,
		Status:                      string(b.Status),
		BuildEnvID:                  b.BuildEnvID,
		GitRepoURL:                  b.GitRepoURL,
		GitRef:                      b.GitRef,
		GitCommitSHA:                b.GitCommitSHA,
		GitCommitMsg:                b.GitCommitMsg,
		ImageFullName:               b.ImageFullName,
		TriggerType:                 string(b.TriggerType),
		TriggeredBy:                 b.TriggeredBy,
		JobName:                     b.JobName,
		JobNamespace:                b.JobNamespace,
		StartedAt:                   b.StartedAt,
		CompletedAt:                 b.CompletedAt,
		Duration:                    b.Duration,
		ErrorMessage:                b.ErrorMessage,
		CreatedAt:                   b.CreatedAt,
	}

	if b.App != nil {
		appResp := ToAppResponse(b.App)
		resp.App = &appResp
	}

	return resp
}

func sanitizeRef(ref string) string {
	ref = strings.ReplaceAll(ref, "/", "-")
	ref = strings.ReplaceAll(ref, " ", "-")
	if len(ref) > 32 {
		ref = ref[:32]
	}
	return ref
}

// ListBuildsByCodeRepository returns builds for a code repository.
func ListBuildsByCodeRepository(repoID string) ([]entities.Build, error) {
	var builds []entities.Build
	if err := db.DB.Where("code_repository_id = ?", repoID).
		Order("build_number DESC").
		Find(&builds).Error; err != nil {
		return nil, err
	}
	return builds, nil
}

func ListDeploymentsByCodeRepository(repoID string) ([]entities.Build, error) {
	var builds []entities.Build
	if err := db.DB.Preload("App.Env").
		Where("code_repository_id = ? AND app_id IS NOT NULL", repoID).
		Order("created_at DESC").
		Find(&builds).Error; err != nil {
		return nil, err
	}
	return builds, nil
}

// GetBuildByCodeRepository returns a build that belongs to the given code repository.
func GetBuildByCodeRepository(repoID, buildID string) (*entities.Build, error) {
	var build entities.Build
	if err := db.DB.Preload("BuildEnv.Cluster").
		Where("id = ? AND code_repository_id = ?", buildID, repoID).
		First(&build).Error; err != nil {
		return nil, err
	}
	return &build, nil
}

// TriggerCodeRepositoryBuild starts a build for a code repository build config (build_config_id + build_env_id required).
func TriggerCodeRepositoryBuild(repoID, userID string, req *models.TriggerCodeRepositoryBuildRequest) (*entities.Build, error) {
	repo, err := GetCodeRepository(repoID)
	if err != nil {
		return nil, fmt.Errorf("code repository not found: %w", err)
	}
	config, err := GetCodeRepositoryBuildConfig(req.BuildConfigID)
	if err != nil {
		return nil, fmt.Errorf("build config not found: %w", err)
	}
	if config.CodeRepositoryID != repoID {
		return nil, errors.New("build config does not belong to this code repository")
	}
	registry, err := GetContainerRegistry(config.RegistryID)
	if err != nil {
		return nil, fmt.Errorf("container registry not found: %w", err)
	}
	buildEnv, err := GetEnv(req.BuildEnvID)
	if err != nil {
		return nil, fmt.Errorf("build environment not found: %w", err)
	}
	if buildEnv.ProjectID != repo.ProjectID {
		return nil, errors.New("build environment must belong to the same project as the code repository")
	}

	var activeCount int64
	if err := db.DB.Model(&entities.Build{}).
		Where("code_repository_id = ? AND code_repository_build_config_id = ? AND status IN ?",
			repoID, config.ID, []entities.BuildStatus{
				entities.BuildStatusPending, entities.BuildStatusCloning, entities.BuildStatusBuilding,
			}).Count(&activeCount).Error; err != nil {
		return nil, err
	}
	if activeCount > 0 {
		return nil, errors.New("an active build already exists for this build config")
	}

	var lastBuild entities.Build
	var buildNumber int
	if err := db.DB.Where("code_repository_id = ?", repoID).Order("build_number DESC").First(&lastBuild).Error; err != nil {
		buildNumber = 1
	} else {
		buildNumber = lastBuild.BuildNumber + 1
	}

	gitRef := config.GitRef
	if req.GitRef != "" {
		gitRef = req.GitRef
	}
	if gitRef == "" {
		gitRef = "main"
	}
	imageTag := fmt.Sprintf("%s-%d", sanitizeRef(gitRef), buildNumber)
	if req.ImageTag != "" {
		imageTag = req.ImageTag
	}
	// Store full image reference (registry/namespace/image:tag; for Docker Hub omit URL) for deploy
	imageFullName := core.BuildImageFullName(registry, config.ImageName, imageTag)

	now := time.Now()
	cfgID := config.ID
	build := &entities.Build{
		Base:                        entities.Base{ID: uuid.New()},
		CodeRepositoryID:            &repoID,
		CodeRepositoryBuildConfigID: &cfgID,
		AppID:                       nil,
		BuildConfigID:               nil,
		BuildNumber:                 buildNumber,
		Status:                      entities.BuildStatusPending,
		BuildEnvID:                  buildEnv.ID,
		GitRepoURL:                  repo.GitRepoURL,
		GitRef:                      gitRef,
		ImageFullName:               imageFullName,
		TriggerType:                 entities.BuildTriggerManual,
		TriggeredBy:                 userID,
		StartedAt:                   &now,
	}

	if req.AutoDeploy != nil && *req.AutoDeploy {
		build.PendingDeployEnvID = req.DeployEnvID
		build.PendingDeployAppID = req.DeployAppID
		build.PendingDeployAppName = req.DeployAppName
		build.PendingDeployAppSlug = req.DeployAppSlug
	}
	if err := db.DB.Create(build).Error; err != nil {
		return nil, err
	}

	jobSlug := CodeRepositorySlugForJob(repo.Name, config.Name)
	jobName, jobNamespace, err := core.SubmitBuildJobFromCodeRepo(
		context.Background(),
		build,
		repo,
		config,
		registry,
		buildEnv,
		jobSlug,
	)
	if err != nil {
		build.Status = entities.BuildStatusFailed
		build.ErrorMessage = err.Error()
		completedAt := time.Now()
		build.CompletedAt = &completedAt
		db.DB.Save(build)
		return nil, fmt.Errorf("failed to submit build job: %w", err)
	}

	build.JobName = jobName
	build.JobNamespace = jobNamespace
	if err := db.DB.Save(build).Error; err != nil {
		return nil, err
	}
	core.GlobalBuildWatcher.StartWatching(build)
	return build, nil
}

// DeployCodeRepositoryBuild deploys a successful build to a target environment (create or update app).
func DeployCodeRepositoryBuild(repoID, buildID string, req *models.DeployCodeRepositoryBuildRequest) (*entities.Build, *entities.App, error) {
	build, err := GetBuildByCodeRepository(repoID, buildID)
	if err != nil {
		return nil, nil, err
	}
	if build.Status != entities.BuildStatusSucceeded {
		return nil, nil, errors.New("can only deploy a successful build")
	}
	if build.ImageFullName == "" {
		return nil, nil, errors.New("build has no image")
	}
	if build.CodeRepositoryBuildConfigID == nil || *build.CodeRepositoryBuildConfigID == "" {
		return nil, nil, errors.New("build has no build config (registry unknown)")
	}

	repo, err := GetCodeRepository(repoID)
	if err != nil {
		return nil, nil, err
	}
	targetEnv, err := GetEnv(req.TargetEnvID)
	if err != nil {
		return nil, nil, fmt.Errorf("target environment not found: %w", err)
	}
	if targetEnv.ProjectID != repo.ProjectID {
		return nil, nil, errors.New("target environment must belong to the same project")
	}

	config, err := GetCodeRepositoryBuildConfig(*build.CodeRepositoryBuildConfigID)
	if err != nil {
		return nil, nil, fmt.Errorf("build config not found: %w", err)
	}
	registry, err := GetContainerRegistry(config.RegistryID)
	if err != nil {
		return nil, nil, err
	}

	var app *entities.App
	if req.AppID != "" {
		app, err = GetApp(req.AppID)
		if err != nil {
			return nil, nil, fmt.Errorf("app not found: %w", err)
		}
		if app.EnvID != req.TargetEnvID {
			return nil, nil, errors.New("app does not belong to the target environment")
		}
		app.ContainerImage = build.ImageFullName
		app.RegistryUsername = registry.Username
		app.RegistryPassword = registry.Password
		rid := repoID
		app.CodeRepositoryID = &rid
		if err := db.DB.Save(app).Error; err != nil {
			return nil, nil, err
		}
	} else {
		if req.Slug == "" || req.Name == "" {
			return nil, nil, errors.New("name and slug are required when creating a new app")
		}
		app, err = CreateAppFromCodeRepositoryBuild(req.TargetEnvID, req.Slug, req.Name, build.ImageFullName, registry.Username, registry.Password, &repoID)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := core.ApplyApp(context.Background(), app); err != nil {
		return nil, nil, fmt.Errorf("failed to deploy: %w", err)
	}
	app.DeployStatus = "deployed"
	db.DB.Model(app).Update("deploy_status", "deployed")

	appID := app.ID
	build.AppID = &appID
	db.DB.Save(build)

	return build, app, nil
}
