package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// BuildWatcher monitors active build jobs and updates their status.
type BuildWatcher struct {
	mu       sync.Mutex
	watching map[string]context.CancelFunc // buildID -> cancel func
}

var GlobalBuildWatcher = &BuildWatcher{
	watching: make(map[string]context.CancelFunc),
}

// StartWatching begins monitoring a build job.
func (bw *BuildWatcher) StartWatching(build *entities.Build) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if _, exists := bw.watching[build.ID]; exists {
		return // Already watching
	}

	ctx, cancel := context.WithCancel(context.Background())
	bw.watching[build.ID] = cancel

	go bw.watchBuild(ctx, build.ID, build.BuildEnvID, build.JobName, build.JobNamespace)
}

// StopWatching stops monitoring a build job.
func (bw *BuildWatcher) StopWatching(buildID string) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if cancel, exists := bw.watching[buildID]; exists {
		cancel()
		delete(bw.watching, buildID)
	}
}

// RecoverActiveBuilds finds and resumes watching for builds that are still active.
func (bw *BuildWatcher) RecoverActiveBuilds() {
	var builds []entities.Build
	if err := db.DB.Where("status IN ?", []entities.BuildStatus{
		entities.BuildStatusPending,
		entities.BuildStatusCloning,
		entities.BuildStatusBuilding,
	}).Find(&builds).Error; err != nil {
		log.Printf("Failed to recover active builds: %v", err)
		return
	}

	for i := range builds {
		log.Printf("Recovering watch for build %s (job: %s)", builds[i].ID, builds[i].JobName)
		bw.StartWatching(&builds[i])
	}
}

func (bw *BuildWatcher) watchBuild(ctx context.Context, buildID, buildEnvID, jobName, jobNamespace string) {
	defer func() {
		bw.mu.Lock()
		delete(bw.watching, buildID)
		bw.mu.Unlock()
	}()

	// Get the build env to find the cluster
	var buildEnv entities.Env
	if err := db.DB.First(&buildEnv, "id = ?", buildEnvID).Error; err != nil {
		log.Printf("Build watcher: failed to get build env %s: %v", buildEnvID, err)
		updateBuildFailed(buildID, "Failed to get build environment")
		return
	}

	client, err := kube.GlobalClusterStore.GetClient(buildEnv.ClusterID)
	if err != nil {
		log.Printf("Build watcher: failed to get cluster client for build %s: %v", buildID, err)
		updateBuildFailed(buildID, "Failed to get cluster client")
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job, err := client.BatchV1().Jobs(jobNamespace).Get(ctx, jobName, metav1.GetOptions{})
			if err != nil {
				log.Printf("Build watcher: failed to get job %s: %v", jobName, err)
				continue
			}

			// Check if job is complete
			if job.Status.Succeeded > 0 {
				now := time.Now()
				var build entities.Build
				if err := db.DB.First(&build, "id = ?", buildID).Error; err != nil {
					log.Printf("Build watcher: failed to get build %s: %v", buildID, err)
					return
				}

				build.Status = entities.BuildStatusSucceeded
				build.CompletedAt = &now
				if build.StartedAt != nil {
					build.Duration = int(now.Sub(*build.StartedAt).Seconds())
				}

				if err := db.DB.Save(&build).Error; err != nil {
					log.Printf("Build watcher: failed to update build %s: %v", buildID, err)
					return
				}

				// Handle auto-deploy
				go handleAutoDeploy(&build)

				// Cleanup secrets
				go CleanupBuildSecrets(context.Background(), buildEnv.ClusterID, buildID, jobNamespace)

				log.Printf("Build %s succeeded", buildID)
				return
			}

			if job.Status.Failed > 0 {
				now := time.Now()
				errMsg := "Build job failed"

				// Try to get error message from pod
				pods, _ := client.CoreV1().Pods(jobNamespace).List(ctx, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("job-name=%s", jobName),
				})
				if pods != nil && len(pods.Items) > 0 {
					pod := pods.Items[0]
					logTail := readBuildFailureLogTail(ctx, client, jobNamespace, pod.Name, pod.Spec.Containers)
					for _, cs := range pod.Status.ContainerStatuses {
						if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
							errMsg = fmt.Sprintf("%s: %s", cs.Name, cs.State.Terminated.Reason)
							if cs.State.Terminated.Message != "" {
								errMsg = fmt.Sprintf("%s - %s", errMsg, cs.State.Terminated.Message)
							}
							break
						}
					}
					errMsg = normalizeBuildFailureMessage(errMsg, logTail)
				}

				var build entities.Build
				if err := db.DB.First(&build, "id = ?", buildID).Error; err != nil {
					return
				}

				build.Status = entities.BuildStatusFailed
				build.CompletedAt = &now
				build.ErrorMessage = errMsg
				if build.StartedAt != nil {
					build.Duration = int(now.Sub(*build.StartedAt).Seconds())
				}
				db.DB.Save(&build)

				// Cleanup secrets
				go CleanupBuildSecrets(context.Background(), buildEnv.ClusterID, buildID, jobNamespace)

				log.Printf("Build %s failed: %s", buildID, errMsg)
				return
			}

			// Update status based on pod phase
			pods, _ := client.CoreV1().Pods(jobNamespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name=%s", jobName),
			})
			if pods != nil && len(pods.Items) > 0 {
				pod := pods.Items[0]
				var newStatus entities.BuildStatus

				switch pod.Status.Phase {
				case "Pending":
					// Check init containers
					for _, cs := range pod.Status.InitContainerStatuses {
						if cs.State.Running != nil {
							newStatus = entities.BuildStatusCloning
							break
						}
					}
				case "Running":
					newStatus = entities.BuildStatusBuilding
				}

				if newStatus != "" {
					var build entities.Build
					if err := db.DB.First(&build, "id = ?", buildID).Error; err == nil {
						if build.Status != newStatus {
							build.Status = newStatus
							if newStatus == entities.BuildStatusBuilding && build.StartedAt == nil {
								now := time.Now()
								build.StartedAt = &now
							}
							db.DB.Save(&build)
						}
					}
				}
			}
		}
	}
}

func readBuildFailureLogTail(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespace, podName string,
	containers []corev1.Container,
) string {
	containerName := ""
	for _, container := range containers {
		if container.Name == "buildctl" {
			containerName = container.Name
			break
		}
	}
	if containerName == "" && len(containers) > 0 {
		containerName = containers[0].Name
	}
	if containerName == "" {
		return ""
	}

	tailLines := int64(40)
	stream, err := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}).Stream(ctx)
	if err != nil {
		return ""
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func normalizeBuildFailureMessage(rawMessage, logTail string) string {
	combined := strings.ToLower(strings.TrimSpace(rawMessage + "\n" + logTail))

	if strings.Contains(combined, "failed to mount") &&
		strings.Contains(combined, "/var/lib/buildkit") &&
		strings.Contains(combined, "operation not permitted") {
		return "BuildKit builder is missing required mount privileges. The shared ketches-buildkitd StatefulSet must run privileged so it can mount snapshot state under /var/lib/buildkit. Reconcile the builder and retry."
	}

	if strings.Contains(combined, "exec format error") ||
		strings.Contains(combined, ".buildkit_qemu_emulator") {
		return "Multi-arch build requires binfmt/QEMU support, but cross-architecture emulation is unavailable. Verify that DaemonSet ketches-buildkit-binfmt is Ready in namespace ketches-build, then retry."
	}

	rawMessage = strings.TrimSpace(rawMessage)
	if rawMessage != "" {
		return rawMessage
	}

	logTail = strings.TrimSpace(logTail)
	if logTail == "" {
		return "Build job failed"
	}

	lines := strings.Split(logTail, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return "Build job failed"
}

func handleAutoDeploy(build *entities.Build) {
	// Check for a pending build deployment record (code-repo auto-deploy)
	var bd entities.BuildDeployment
	pendingDeployErr := db.DB.Where("build_id = ? AND status = ?",
		build.ID, entities.BuildDeploymentStatusPending).First(&bd).Error

	if pendingDeployErr == nil && bd.EnvID != "" {
		// This build has a pending deployment — it was a code-repo build with auto-deploy
		handleCodeRepoBuildDeploy(build, &bd)
		return
	}

	// App-scoped build: check if build setting has auto_deploy enabled
	if build.BuildSettingID == "" {
		return
	}
	var setting entities.BuildSetting
	if err := db.DB.First(&setting, "id = ?", build.BuildSettingID).Error; err != nil {
		log.Printf("Auto deploy: failed to get build setting: %v", err)
		return
	}
	var registry entities.ContainerRegistry
	if err := db.DB.First(&registry, "id = ?", setting.RegistryID).Error; err != nil {
		log.Printf("Auto deploy: failed to get registry: %v", err)
		return
	}

	// Find the BuildDeployment for this app-scoped build
	var appBD entities.BuildDeployment
	if err := db.DB.Where("build_id = ? AND status = ?", build.ID, entities.BuildDeploymentStatusPending).
		Order("created_at DESC").
		First(&appBD).Error; err != nil || appBD.AppID == nil || *appBD.AppID == "" {
		return
	}

	var app entities.App
	if err := db.DB.First(&app, "id = ?", *appBD.AppID).Error; err != nil {
		log.Printf("Auto deploy: failed to get app: %v", err)
		markBuildDeploymentFailed(&appBD, "failed to get app")
		return
	}
	var env entities.Env
	if err := db.DB.First(&env, "id = ?", app.EnvID).Error; err != nil {
		log.Printf("Auto deploy: failed to get env: %v", err)
		markBuildDeploymentFailed(&appBD, "failed to get env")
		return
	}
	var cluster entities.Cluster
	if err := db.DB.First(&cluster, "id = ?", env.ClusterID).Error; err != nil {
		log.Printf("Auto deploy: failed to get cluster: %v", err)
		markBuildDeploymentFailed(&appBD, "failed to get cluster")
		return
	}

	app.ContainerImage = build.ImageFullName
	app.RegistryUsername = registry.Username
	app.RegistryPassword = registry.Password
	if err := db.DB.Save(&app).Error; err != nil {
		log.Printf("Auto deploy: failed to update app image: %v", err)
		markBuildDeploymentFailed(&appBD, "failed to update app image")
		return
	}

	var project entities.Project
	if err := db.DB.First(&project, "id = ?", env.ProjectID).Error; err != nil {
		log.Printf("Auto deploy: failed to get project: %v", err)
		markBuildDeploymentFailed(&appBD, "failed to get project")
		return
	}

	appCtx := models.AppContext{
		App: app,
		EnvContext: models.EnvContext{
			Env:     env,
			Project: project,
			Cluster: cluster,
		},
	}
	now := time.Now()
	if err := ApplyApp(context.Background(), &appCtx); err != nil {
		log.Printf("Auto deploy: failed to apply app: %v", err)
		db.DB.Model(&appBD).Updates(map[string]any{
			"status":        entities.BuildDeploymentStatusFailed,
			"error_message": err.Error(),
			"deployed_at":   &now,
		})
		return
	}

	app.DeployStatus = "deployed"
	db.DB.Model(&app).Update("deploy_status", "deployed")

	db.DB.Model(&appBD).Updates(map[string]any{
		"status":      entities.BuildDeploymentStatusDeployed,
		"deployed_at": &now,
	})
	log.Printf("Auto deploy: successfully deployed build %s to app %s", build.ID, app.ID)
}

func handleCodeRepoBuildDeploy(build *entities.Build, bd *entities.BuildDeployment) {
	var deployEnv entities.Env
	if err := db.DB.First(&deployEnv, "id = ?", bd.EnvID).Error; err != nil {
		log.Printf("Code repo auto-deploy: failed to get deploy env: %v", err)
		markBuildDeploymentFailed(bd, "failed to get deploy environment")
		return
	}
	var deployCluster entities.Cluster
	if err := db.DB.First(&deployCluster, "id = ?", deployEnv.ClusterID).Error; err != nil {
		log.Printf("Code repo auto-deploy: failed to get cluster: %v", err)
		markBuildDeploymentFailed(bd, "failed to get deploy cluster")
		return
	}

	var setting entities.BuildSetting
	if err := db.DB.First(&setting, "id = ?", build.BuildSettingID).Error; err != nil {
		log.Printf("Code repo auto-deploy: failed to get build setting: %v", err)
		markBuildDeploymentFailed(bd, "failed to get build setting")
		return
	}
	var repoRegistry entities.ContainerRegistry
	if err := db.DB.First(&repoRegistry, "id = ?", setting.RegistryID).Error; err != nil {
		log.Printf("Code repo auto-deploy: failed to get registry: %v", err)
		markBuildDeploymentFailed(bd, "failed to get registry")
		return
	}

	var app *entities.App
	if bd.AppID != nil && *bd.AppID != "" {
		var existingApp entities.App
		if err := db.DB.First(&existingApp, "id = ?", *bd.AppID).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to get app: %v", err)
			markBuildDeploymentFailed(bd, "failed to get app")
			return
		}
		var appEnv entities.Env
		if err := db.DB.First(&appEnv, "id = ?", existingApp.EnvID).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to get app env: %v", err)
			markBuildDeploymentFailed(bd, "failed to get app env")
			return
		}
		var appCluster entities.Cluster
		if err := db.DB.First(&appCluster, "id = ?", appEnv.ClusterID).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to get app cluster: %v", err)
			markBuildDeploymentFailed(bd, "failed to get app cluster")
			return
		}
		app = &existingApp

		app.ContainerImage = build.ImageFullName
		app.RegistryUsername = repoRegistry.Username
		app.RegistryPassword = repoRegistry.Password
		if err := db.DB.Save(app).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to update app: %v", err)
			markBuildDeploymentFailed(bd, "failed to update app image")
			return
		}

		var project entities.Project
		if err := db.DB.First(&project, "id = ?", appEnv.ProjectID).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to get project: %v", err)
			markBuildDeploymentFailed(bd, "failed to get project")
			return
		}

		appCtx := models.AppContext{
			App: *app,
			EnvContext: models.EnvContext{
				Env:     appEnv,
				Project: project,
				Cluster: appCluster,
			},
		}
		if err := ApplyApp(context.Background(), &appCtx); err != nil {
			log.Printf("Code repo auto-deploy: failed to apply app: %v", err)
			markBuildDeploymentFailed(bd, err.Error())
			return
		}

		app.DeployStatus = "deployed"
		db.DB.Model(app).Update("deploy_status", "deployed")

	} else if bd.AppName != "" && bd.AppSlug != "" {
		repoID := setting.CodeRepositoryID
		newApp := &entities.App{
			Base:             entities.Base{ID: uuid.New()},
			Slug:             bd.AppSlug,
			Name:             bd.AppName,
			EnvID:            deployEnv.ID,
			ContainerImage:   build.ImageFullName,
			RegistryUsername: repoRegistry.Username,
			RegistryPassword: repoRegistry.Password,
			Replicas:         1,
			RequestCPU:       100,
			RequestMemory:    128,
			LimitCPU:         1000,
			LimitMemory:      512,
			AppType:          "Deployment",
			DeployStatus:     "undeployed",
			CodeRepositoryID: repoID,
		}
		if err := db.DB.Create(newApp).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to create app: %v", err)
			markBuildDeploymentFailed(bd, "failed to create app")
			return
		}

		var project entities.Project
		if err := db.DB.First(&project, "id = ?", deployEnv.ProjectID).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to get project: %v", err)
			markBuildDeploymentFailed(bd, "failed to get project")
			return
		}

		appCtx := models.AppContext{
			App: *newApp,
			EnvContext: models.EnvContext{
				Env:     deployEnv,
				Project: project,
				Cluster: deployCluster,
			},
		}
		if err := ApplyApp(context.Background(), &appCtx); err != nil {
			log.Printf("Code repo auto-deploy: failed to apply new app: %v", err)
			markBuildDeploymentFailed(bd, err.Error())
			return
		}

		newApp.DeployStatus = "deployed"
		db.DB.Model(newApp).Update("deploy_status", "deployed")
		app = newApp

		// Update the BuildDeployment with the newly created app ID
		db.DB.Model(bd).Update("app_id", newApp.ID)
	} else {
		log.Printf("Code repo auto-deploy: missing app deployment info in build deployment record")
		markBuildDeploymentFailed(bd, "missing app deployment info")
		return
	}

	now := time.Now()
	db.DB.Model(bd).Updates(map[string]any{
		"status":      entities.BuildDeploymentStatusDeployed,
		"deployed_at": &now,
	})
	log.Printf("Code repo auto-deploy: successfully deployed build %s to app %s", build.ID, app.ID)
}

func markBuildDeploymentFailed(bd *entities.BuildDeployment, errMsg string) {
	now := time.Now()
	db.DB.Model(bd).Updates(map[string]any{
		"status":        entities.BuildDeploymentStatusFailed,
		"error_message": errMsg,
		"deployed_at":   &now,
	})
}

func updateBuildFailed(buildID, errMsg string) {
	now := time.Now()
	db.DB.Model(&entities.Build{}).Where("id = ?", buildID).Updates(map[string]any{
		"status":        entities.BuildStatusFailed,
		"completed_at":  &now,
		"error_message": errMsg,
	})
}
