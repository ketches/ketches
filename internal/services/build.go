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

func ListAppBuilds(appID string, page, pageSize int) (int64, []models.AppBuildResponse, error) {
	var total int64
	var builds []models.AppBuildResponse
	query := db.DB.Model(&entities.Build{}).
		Where(`
			EXISTS (
				SELECT 1
				FROM build_deployments bd
				WHERE bd.build_id = builds.id
				  AND bd.app_id = ?
			)
		`, appID)
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.
		Select(`
			builds.id AS id,
			'' AS build_setting_name,
			builds.build_setting_id AS build_setting_id,
			builds.build_number AS build_number,
			builds.status AS status,
			COALESCE(latest_bd.status, '') AS deploy_status,
			builds.build_env_id AS build_env_id,
			COALESCE(builds.git_repo_url, '') AS git_repo_url,
			COALESCE(builds.git_ref, '') AS git_ref,
			COALESCE(builds.git_commit_sha, '') AS git_commit_sha,
			COALESCE(builds.git_commit_msg, '') AS git_commit_msg,
			COALESCE(builds.image_full_name, '') AS image_full_name,
			COALESCE(builds.trigger_type, '') AS trigger_type,
			COALESCE(builds.triggered_by, '') AS triggered_by,
			COALESCE(builds.job_name, '') AS job_name,
			COALESCE(builds.job_namespace, '') AS job_namespace,
			builds.started_at AS started_at,
			builds.completed_at AS completed_at,
			builds.duration AS duration,
			COALESCE(builds.error_message, '') AS error_message,
			COALESCE(latest_bd.error_message, '') AS deployment_error_message,
			builds.created_at AS created_at
		`).
		Joins(`
			LEFT JOIN build_deployments latest_bd ON latest_bd.id = (
				SELECT bd2.id
				FROM build_deployments bd2
				WHERE bd2.build_id = builds.id AND bd2.app_id = ?
				ORDER BY bd2.created_at DESC
				LIMIT 1
			)
		`, appID).
		Order("builds.build_number DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&builds).Error; err != nil {
		return 0, nil, err
	}
	return total, builds, nil
}

func GetBuild(buildID string) (*entities.Build, error) {
	var build entities.Build
	if err := db.DB.First(&build, "id = ?", buildID).Error; err != nil {
		return nil, err
	}
	return &build, nil
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
		var buildEnv entities.Env
		if err := db.DB.First(&buildEnv, "id = ?", build.BuildEnvID).Error; err != nil {
			return nil, err
		}
		if err := core.CancelBuildJob(
			context.Background(),
			buildEnv.ClusterID,
			build.JobName,
			build.JobNamespace,
		); err != nil {
			log.Printf("Failed to cancel build job: %v", err)
		}
	}

	// Cleanup secrets
	if buildEnv, err := GetEnv(build.BuildEnvID); err == nil {
		go core.CleanupBuildSecrets(context.Background(), buildEnv.ClusterID, buildID, build.JobNamespace)
	}

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

func DeployBuild(ctx context.Context, buildID string) (*entities.Build, error) {
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

	if build.BuildSettingID == "" {
		return nil, errors.New("build has no build setting")
	}
	var setting entities.BuildSetting
	if err := db.DB.First(&setting, "id = ?", build.BuildSettingID).Error; err != nil {
		return nil, err
	}
	var registry entities.ContainerRegistry
	if err := db.DB.First(&registry, "id = ?", setting.RegistryID).Error; err != nil {
		return nil, err
	}

	var bd entities.BuildDeployment
	if err := db.DB.Where("build_id = ?", buildID).
		Order("created_at DESC").
		First(&bd).Error; err != nil {
		return nil, errors.New("build has no associated app; use code repository deploy for this build")
	}
	if bd.AppID == nil || *bd.AppID == "" {
		return nil, errors.New("build has no associated app; use code repository deploy for this build")
	}
	appCtx, err := GetAppContext(ctx, *bd.AppID)
	if err != nil {
		return nil, err
	}

	// Update app's container image
	appCtx.App.ContainerImage = build.ImageFullName
	appCtx.App.RegistryUsername = registry.Username
	appCtx.App.RegistryPassword = registry.Password

	if err := db.DB.Save(&appCtx.App).Error; err != nil {
		return nil, err
	}

	// Apply the app
	if err := core.ApplyApp(ctx, appCtx); err != nil {
		now := time.Now()
		db.DB.Model(&bd).Updates(map[string]any{
			"status":        entities.BuildDeploymentStatusFailed,
			"error_message": err.Error(),
			"deployed_at":   &now,
		})
		return nil, fmt.Errorf("failed to deploy: %w", err)
	}

	// Update deploy status
	appCtx.App.DeployStatus = "deployed"
	db.DB.Model(&appCtx.App).Update("deploy_status", "deployed")

	now := time.Now()
	db.DB.Model(&bd).Updates(map[string]any{
		"status":      entities.BuildDeploymentStatusDeployed,
		"deployed_at": &now,
	})
	return build, nil
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

	buildEnv, err := GetEnv(build.BuildEnvID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to load build environment"})
		return
	}
	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
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

func ToBuildResponse(c context.Context, b *entities.Build) models.BuildResponse {
	resp := models.BuildResponse{
		ID:             b.ID,
		BuildSettingID: b.BuildSettingID,
		BuildNumber:    b.BuildNumber,
		Status:         string(b.Status),
		BuildEnvID:     b.BuildEnvID,
		GitRepoURL:     b.GitRepoURL,
		GitRef:         b.GitRef,
		GitCommitSHA:   b.GitCommitSHA,
		GitCommitMsg:   b.GitCommitMsg,
		ImageFullName:  b.ImageFullName,
		TriggerType:    string(b.TriggerType),
		TriggeredBy:    derefBuildString(b.TriggeredBy),
		JobName:        b.JobName,
		JobNamespace:   b.JobNamespace,
		StartedAt:      b.StartedAt,
		CompletedAt:    b.CompletedAt,
		Duration:       b.Duration,
		ErrorMessage:   b.ErrorMessage,
		CreatedAt:      b.CreatedAt,
	}

	return resp
}

func derefBuildString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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
	if err := db.DB.
		Joins("JOIN build_settings ON build_settings.id = builds.build_setting_id").
		Where("build_settings.code_repository_id = ?", repoID).
		Order("builds.build_number DESC").
		Find(&builds).Error; err != nil {
		return nil, err
	}
	return builds, nil
}

type codeRepositoryDeploymentRow struct {
	BuildID                string `gorm:"column:build_id"`
	BuildSettingID         string `gorm:"column:build_setting_id"`
	BuildNumber            int    `gorm:"column:build_number"`
	GitRef                 string `gorm:"column:git_ref"`
	ImageFullName          string `gorm:"column:image_full_name"`
	BuildCreatedAt         time.Time
	DeploymentID           string  `gorm:"column:deployment_id"`
	DeploymentStatus       string  `gorm:"column:deployment_status"`
	DeploymentErrorMessage string  `gorm:"column:deployment_error_message"`
	AppID                  *string `gorm:"column:app_id"`
	AppName                string  `gorm:"column:app_name"`
	EnvID                  string  `gorm:"column:env_id"`
	EnvName                string  `gorm:"column:env_name"`
}

func toCodeRepositoryDeploymentResponse(row *codeRepositoryDeploymentRow) models.CodeRepositoryDeploymentResponse {
	resp := models.CodeRepositoryDeploymentResponse{
		ID:             row.BuildID,
		DeploymentID:   row.DeploymentID,
		BuildSettingID: row.BuildSettingID,
		BuildNumber:    row.BuildNumber,
		Status:         row.DeploymentStatus,
		GitRef:         row.GitRef,
		ImageFullName:  row.ImageFullName,
		ErrorMessage:   row.DeploymentErrorMessage,
		CreatedAt:      row.BuildCreatedAt,
		AppID:          stringValue(row.AppID),
		AppName:        row.AppName,
		EnvName:        row.EnvName,
	}

	return resp
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func ListDeployments(repoID string) ([]models.CodeRepositoryDeploymentResponse, error) {
	const deploymentListSelectCols = `
		b.id AS build_id,
		b.build_setting_id AS build_setting_id,
		b.build_number AS build_number,
		b.git_ref AS git_ref,
		b.image_full_name AS image_full_name,
		b.created_at AS build_created_at,
		bd.id AS deployment_id,
		bd.status AS deployment_status,
		bd.error_message AS deployment_error_message,
		bd.app_id AS app_id,
		COALESCE(a.name, bd.app_name) AS app_name,
		bd.env_id AS env_id,
		COALESCE(e.name, '') AS env_name`

	var rows []codeRepositoryDeploymentRow
	if err := db.DB.Table("build_deployments bd").
		Select(deploymentListSelectCols).
		Joins("JOIN builds b ON b.id = bd.build_id").
		Joins("JOIN build_settings bs ON bs.id = b.build_setting_id").
		Joins("LEFT JOIN apps a ON a.id = bd.app_id").
		Joins("LEFT JOIN envs e ON e.id = bd.env_id").
		Where("bs.code_repository_id = ?", repoID).
		Order("bd.created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	res := make([]models.CodeRepositoryDeploymentResponse, 0, len(rows))
	for i := range rows {
		res = append(res, toCodeRepositoryDeploymentResponse(&rows[i]))
	}
	return res, nil
}

func ListDeployedAppsByEnvironmentAndBuildSetting(envID, buildSettingID string) ([]models.SimpleApp, error) {
	var apps []models.SimpleApp
	if err := db.DB.Table("build_deployments bd").
		Select("a.id AS id, a.name AS name").
		Joins("JOIN apps a ON a.id = bd.app_id").
		Joins("JOIN builds b ON b.id = bd.build_id").
		Where("b.build_setting_id = ?", buildSettingID).
		Where("bd.env_id = ?", envID).
		Where("bd.app_id IS NOT NULL").
		Order("a.created_at DESC").
		Group("a.id, a.name, a.created_at").
		Find(&apps).Error; err != nil {
		return nil, err
	}

	return apps, nil
}

func TriggerCodeRepositoryBuild(repoID, userID string, req *models.TriggerCodeRepositoryBuildRequest) (*entities.Build, error) {
	repo, err := GetCodeRepository(repoID)
	if err != nil {
		return nil, fmt.Errorf("code repository not found: %w", err)
	}
	setting, err := GetBuildSetting(req.BuildSettingID)
	if err != nil {
		return nil, fmt.Errorf("build setting not found: %w", err)
	}
	if setting.CodeRepositoryID == nil || *setting.CodeRepositoryID != repoID {
		return nil, errors.New("build setting does not belong to this code repository")
	}
	registry, err := GetContainerRegistry(setting.RegistryID)
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

	project, err := GetProject(repo.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	var activeCount int64
	if err := db.DB.Model(&entities.Build{}).
		Where("build_setting_id = ? AND status IN ?",
			setting.ID, []entities.BuildStatus{
				entities.BuildStatusPending, entities.BuildStatusCloning, entities.BuildStatusBuilding,
			}).Count(&activeCount).Error; err != nil {
		return nil, err
	}
	if activeCount > 0 {
		return nil, errors.New("an active build already exists for this build setting")
	}

	var lastBuild entities.Build
	var buildNumber int
	if err := db.DB.
		Joins("JOIN build_settings ON build_settings.id = builds.build_setting_id").
		Where("build_settings.code_repository_id = ?", repoID).
		Order("builds.build_number DESC").
		First(&lastBuild).Error; err != nil {
		buildNumber = 1
	} else {
		buildNumber = lastBuild.BuildNumber + 1
	}

	gitRef := setting.GitRef
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
	imageFullName := core.BuildImageFullName(registry, setting.ImageName, imageTag)

	now := time.Now()
	triggeredBy := userID
	var triggeredByPtr *string
	if triggeredBy != "" {
		triggeredByPtr = &triggeredBy
	}
	build := &entities.Build{
		ID:             uuid.New(),
		BuildSettingID: setting.ID,
		BuildNumber:    buildNumber,
		Status:         entities.BuildStatusPending,
		BuildEnvID:     buildEnv.ID,
		GitRepoURL:     repo.GitRepoURL,
		GitRef:         gitRef,
		ImageFullName:  imageFullName,
		TriggerType:    entities.BuildTriggerManual,
		TriggeredBy:    triggeredByPtr,
		StartedAt:      &now,
	}

	if err := db.DB.Create(build).Error; err != nil {
		return nil, err
	}

	if req.AutoDeploy != nil && *req.AutoDeploy && req.DeployEnvID != "" {
		var appIDPtr *string
		if req.DeployAppID != "" {
			appIDPtr = &req.DeployAppID
		}
		buildDeployment := &entities.BuildDeployment{
			ID:         uuid.New(),
			BuildID:    build.ID,
			AppID:      appIDPtr,
			EnvID:      req.DeployEnvID,
			AppName:    req.DeployAppName,
			AppSlug:    req.DeployAppSlug,
			Status:     entities.BuildDeploymentStatusPending,
			DeployedBy: "auto",
		}
		if err := db.DB.Create(buildDeployment).Error; err != nil {
			log.Printf("TriggerCodeRepositoryBuild: failed to create build deployment record: %v", err)
		}
	}

	jobSlug := CodeRepositorySlugForJob(repo.Name, setting.Name)
	baseRepo := repo.CodeRepository
	jobName, jobNamespace, err := core.SubmitBuildJobFromCodeRepo(
		context.Background(),
		build,
		&baseRepo,
		&setting.BuildSetting,
		registry,
		buildEnv,
		project,
		jobSlug,
	)
	if err != nil {
		core.MarkBuildFailed(build.ID, err.Error())
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
func DeployCodeRepositoryBuild(ctx context.Context, repoID, buildID string, req *models.DeployCodeRepositoryBuildRequest) (*entities.Build, *models.AppContext, error) {
	build, err := GetBuild(buildID)
	if err != nil {
		return nil, nil, err
	}
	if build.Status != entities.BuildStatusSucceeded {
		return nil, nil, errors.New("can only deploy a successful build")
	}
	if build.ImageFullName == "" {
		return nil, nil, errors.New("build has no image")
	}
	if build.BuildSettingID == "" {
		return nil, nil, errors.New("build has no build setting (registry unknown)")
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

	setting, err := GetBuildSetting(build.BuildSettingID)
	if err != nil {
		return nil, nil, fmt.Errorf("build setting not found: %w", err)
	}
	registry, err := GetContainerRegistry(setting.RegistryID)
	if err != nil {
		return nil, nil, err
	}

	var appCtx *models.AppContext
	if req.AppID != "" {
		appCtx, err = GetAppContext(ctx, req.AppID)
		if err != nil {
			return nil, nil, fmt.Errorf("app not found: %w", err)
		}
		if appCtx.App.EnvID != req.TargetEnvID {
			return nil, nil, errors.New("app does not belong to the target environment")
		}
		appCtx.App.ContainerImage = build.ImageFullName
		appCtx.App.RegistryUsername = registry.Username
		appCtx.App.RegistryPassword = registry.Password
		repoIDCopy := repoID
		appCtx.App.CodeRepositoryID = &repoIDCopy
		if err := db.DB.Save(&appCtx.App).Error; err != nil {
			return nil, nil, err
		}
	} else {
		if req.Slug == "" || req.Name == "" {
			return nil, nil, errors.New("name and slug are required when creating a new app")
		}
		appCtx, err = CreateAppFromCodeRepositoryBuild(ctx, req.TargetEnvID, req.Slug, req.Name, build.ImageFullName, registry.Username, registry.Password, repoID)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := core.ApplyApp(ctx, appCtx); err != nil {
		return nil, nil, fmt.Errorf("failed to deploy: %w", err)
	}
	appCtx.App.DeployStatus = "deployed"
	db.DB.Model(&appCtx.App).Update("deploy_status", "deployed")

	now := time.Now()
	appIDVal := appCtx.App.ID
	bd := &entities.BuildDeployment{
		ID:         uuid.New(),
		BuildID:    build.ID,
		AppID:      &appIDVal,
		EnvID:      req.TargetEnvID,
		Status:     entities.BuildDeploymentStatusDeployed,
		DeployedBy: "manual",
		DeployedAt: &now,
	}
	db.DB.Create(bd)

	return build, appCtx, nil
}
