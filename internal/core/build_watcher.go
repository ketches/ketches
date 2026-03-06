package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					for _, cs := range pod.Status.ContainerStatuses {
						if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
							errMsg = fmt.Sprintf("%s: %s", cs.Name, cs.State.Terminated.Reason)
							if cs.State.Terminated.Message != "" {
								errMsg = fmt.Sprintf("%s - %s", errMsg, cs.State.Terminated.Message)
							}
							break
						}
					}
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

func handleAutoDeploy(build *entities.Build) {
	if build.PendingDeployEnvID != "" {
		handleCodeRepoBuildDeploy(build)
		return
	}

	if build.BuildConfigID == nil || *build.BuildConfigID == "" {
		return
	}
	var config entities.AppBuildConfig
	if err := db.DB.First(&config, "id = ?", *build.BuildConfigID).Error; err != nil {
		log.Printf("Auto deploy: failed to get build config: %v", err)
		return
	}
	var registry entities.ContainerRegistry
	if err := db.DB.First(&registry, "id = ?", config.RegistryID).Error; err != nil {
		log.Printf("Auto deploy: failed to get registry: %v", err)
		return
	}
	config.Registry = registry

	if !config.AutoDeploy {
		return
	}
	if build.AppID == nil || *build.AppID == "" {
		return
	}

	var app entities.App
	if err := db.DB.First(&app, "id = ?", *build.AppID).Error; err != nil {
		log.Printf("Auto deploy: failed to get app: %v", err)
		return
	}
	var env entities.Env
	if err := db.DB.First(&env, "id = ?", app.EnvID).Error; err != nil {
		log.Printf("Auto deploy: failed to get env: %v", err)
		return
	}
	var cluster entities.Cluster
	if err := db.DB.First(&cluster, "id = ?", env.ClusterID).Error; err != nil {
		log.Printf("Auto deploy: failed to get cluster: %v", err)
		return
	}

	app.ContainerImage = build.ImageFullName
	app.RegistryUsername = config.Registry.Username
	app.RegistryPassword = config.Registry.Password

	if err := db.DB.Save(&app).Error; err != nil {
		log.Printf("Auto deploy: failed to update app image: %v", err)
		return
	}

	appCtx := models.AppContext{
		App:     app,
		Env:     env,
		Cluster: cluster,
	}
	if err := ApplyApp(context.Background(), &appCtx); err != nil {
		log.Printf("Auto deploy: failed to apply app: %v", err)
		return
	}

	app.DeployStatus = "deployed"
	db.DB.Model(&app).Update("deploy_status", "deployed")

	log.Printf("Auto deploy: successfully deployed build %s to app %s", build.ID, app.ID)
}

func handleCodeRepoBuildDeploy(build *entities.Build) {
	var deployEnv entities.Env
	if err := db.DB.First(&deployEnv, "id = ?", build.PendingDeployEnvID).Error; err != nil {
		log.Printf("Code repo auto-deploy: failed to get deploy env: %v", err)
		return
	}
	var deployCluster entities.Cluster
	if err := db.DB.First(&deployCluster, "id = ?", deployEnv.ClusterID).Error; err != nil {
		log.Printf("Code repo auto-deploy: failed to get cluster: %v", err)
		return
	}

	var config entities.CodeRepositoryBuildConfig
	if err := db.DB.First(&config, "id = ?", *build.CodeRepositoryBuildConfigID).Error; err != nil {
		log.Printf("Code repo auto-deploy: failed to get build config: %v", err)
		return
	}
	var repoRegistry entities.ContainerRegistry
	if err := db.DB.First(&repoRegistry, "id = ?", config.RegistryID).Error; err != nil {
		log.Printf("Code repo auto-deploy: failed to get registry: %v", err)
		return
	}
	config.Registry = repoRegistry

	var app *entities.App
	if build.PendingDeployAppID != "" {
		var existingApp entities.App
		if err := db.DB.First(&existingApp, "id = ?", build.PendingDeployAppID).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to get app: %v", err)
			return
		}
		var appEnv entities.Env
		if err := db.DB.First(&appEnv, "id = ?", existingApp.EnvID).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to get app env: %v", err)
			return
		}
		var appCluster entities.Cluster
		if err := db.DB.First(&appCluster, "id = ?", appEnv.ClusterID).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to get app cluster: %v", err)
			return
		}
		app = &existingApp

		app.ContainerImage = build.ImageFullName
		app.RegistryUsername = config.Registry.Username
		app.RegistryPassword = config.Registry.Password

		if err := db.DB.Save(app).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to update app: %v", err)
			return
		}

		appCtx := models.AppContext{
			App:     existingApp,
			Env:     appEnv,
			Cluster: appCluster,
		}
		if err := ApplyApp(context.Background(), &appCtx); err != nil {
			log.Printf("Code repo auto-deploy: failed to apply app: %v", err)
			return
		}

		app.DeployStatus = "deployed"
		db.DB.Model(app).Update("deploy_status", "deployed")
	} else if build.PendingDeployAppName != "" && build.PendingDeployAppSlug != "" {
		newApp := &entities.App{
			Base:             entities.Base{ID: uuid.New()},
			Slug:             build.PendingDeployAppSlug,
			Name:             build.PendingDeployAppName,
			EnvID:            deployEnv.ID,
			ContainerImage:   build.ImageFullName,
			RegistryUsername: config.Registry.Username,
			RegistryPassword: config.Registry.Password,
			Replicas:         1,
			RequestCPU:       100,
			RequestMemory:    128,
			LimitCPU:         1000,
			LimitMemory:      512,
			AppType:          "Deployment",
			DeployStatus:     "undeployed",
			CodeRepositoryID: *build.CodeRepositoryID,
		}
		if err := db.DB.Create(newApp).Error; err != nil {
			log.Printf("Code repo auto-deploy: failed to create app: %v", err)
			return
		}

		appCtx := models.AppContext{
			App:     *newApp,
			Env:     deployEnv,
			Cluster: deployCluster,
		}
		if err := ApplyApp(context.Background(), &appCtx); err != nil {
			log.Printf("Code repo auto-deploy: failed to apply new app: %v", err)
			return
		}

		newApp.DeployStatus = "deployed"
		db.DB.Model(newApp).Update("deploy_status", "deployed")

		app = newApp
	} else {
		log.Printf("Code repo auto-deploy: missing app deployment info")
		return
	}

	build.AppID = &app.ID
	db.DB.Model(build).Update("app_id", app.ID)

	log.Printf("Code repo auto-deploy: successfully deployed build %s to app %s", build.ID, app.ID)
}

func updateBuildFailed(buildID, errMsg string) {
	now := time.Now()
	db.DB.Model(&entities.Build{}).Where("id = ?", buildID).Updates(map[string]any{
		"status":        entities.BuildStatusFailed,
		"completed_at":  &now,
		"error_message": errMsg,
	})
}
