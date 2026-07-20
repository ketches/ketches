package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNormalizeBuildFailureMessage_ExplainsBuildkitPrivilegeErrors(t *testing.T) {
	raw := "buildctl: Error"
	logTail := `error: failed to solve: failed to read dockerfile: failed to mount /tmp/buildkit-mount1111380470: [{Type:bind Source:/var/lib/buildkit/runc-native/snapshots/snapshots/1 Target: Options:[rbind ro]}]: mount source: "/var/lib/buildkit/runc-native/snapshots/snapshots/1", target: "/tmp/buildkit-mount1111380470", fstype: bind, flags: 20481, data: "", err: operation not permitted`

	msg := normalizeBuildFailureMessage(raw, logTail)

	if !strings.Contains(msg, "BuildKit builder is missing required mount privileges") {
		t.Fatalf("expected privileged mount guidance, got %q", msg)
	}
	if !strings.Contains(msg, "ketches-buildkitd") {
		t.Fatalf("expected buildkitd reference, got %q", msg)
	}
}

func TestNormalizeBuildFailureMessage_ExplainsCrossArchExecutionFailures(t *testing.T) {
	raw := "buildctl: Error"
	logTail := `#19 0.210 exec /bin/sh: exec format error`

	msg := normalizeBuildFailureMessage(raw, logTail)

	if !strings.Contains(msg, "Multi-arch build requires binfmt/QEMU support") {
		t.Fatalf("expected binfmt guidance, got %q", msg)
	}
	if !strings.Contains(msg, "ketches-buildkit-binfmt") {
		t.Fatalf("expected binfmt daemonset reference, got %q", msg)
	}
}

func setupBuildWatcherTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/build-watcher.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.Env{},
		&entities.App{},
		&entities.CodeRepository{},
		&entities.BuildSetting{},
		&entities.Build{},
		&entities.BuildDeployment{},
	))

	db.DB = testDB
}

func TestHandleCodeRepoBuildDeployRejectsCrossProjectEnvironment(t *testing.T) {
	setupBuildWatcherTestDB(t)

	repo := entities.CodeRepository{
		Base:       entities.Base{ID: "repo-1"},
		ProjectID:  "project-repo",
		Name:       "Repo",
		Slug:       "repo",
		GitRepoURL: "https://example.com/repo.git",
	}
	require.NoError(t, db.DB.Create(&repo).Error)
	repoID := repo.ID
	setting := entities.BuildSetting{
		ID:               "setting-1",
		CodeRepositoryID: &repoID,
		ImageName:        "repo/app",
		RegistryID:       "registry-1",
	}
	require.NoError(t, db.DB.Create(&setting).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "deploy-env"},
		Slug:      "deploy",
		Name:      "Deploy",
		ProjectID: "project-other",
		ClusterID: "cluster-1",
	}).Error)
	build := entities.Build{
		ID:             "build-1",
		BuildSettingID: setting.ID,
		BuildEnvID:     "build-env",
		BuildNumber:    1,
		Status:         entities.BuildStatusSucceeded,
		ImageFullName:  "registry.example.com/repo/app:new",
	}
	require.NoError(t, db.DB.Create(&build).Error)
	deployment := entities.BuildDeployment{
		ID:         "deployment-1",
		BuildID:    build.ID,
		EnvID:      "deploy-env",
		AppName:    "New App",
		AppSlug:    "new-app",
		Status:     entities.BuildDeploymentStatusPending,
		DeployedBy: "auto",
	}
	require.NoError(t, db.DB.Create(&deployment).Error)

	handleCodeRepoBuildDeploy(&build, &deployment)

	var updatedDeployment entities.BuildDeployment
	require.NoError(t, db.DB.First(&updatedDeployment, "id = ?", deployment.ID).Error)
	assert.Equal(t, entities.BuildDeploymentStatusFailed, updatedDeployment.Status)
	assert.Equal(t, "deploy environment must belong to the same project as the code repository", updatedDeployment.ErrorMessage)

	var appCount int64
	require.NoError(t, db.DB.Model(&entities.App{}).Count(&appCount).Error)
	assert.Zero(t, appCount)
}

func TestHandleCodeRepoBuildDeployRejectsAppOutsideDeployEnvironment(t *testing.T) {
	setupBuildWatcherTestDB(t)

	repo := entities.CodeRepository{
		Base:       entities.Base{ID: "repo-1"},
		ProjectID:  "project-repo",
		Name:       "Repo",
		Slug:       "repo",
		GitRepoURL: "https://example.com/repo.git",
	}
	require.NoError(t, db.DB.Create(&repo).Error)
	repoID := repo.ID
	setting := entities.BuildSetting{
		ID:               "setting-1",
		CodeRepositoryID: &repoID,
		ImageName:        "repo/app",
		RegistryID:       "registry-1",
	}
	require.NoError(t, db.DB.Create(&setting).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "deploy-env"},
		Slug:      "deploy",
		Name:      "Deploy",
		ProjectID: "project-repo",
		ClusterID: "cluster-1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "other-env"},
		Slug:      "other",
		Name:      "Other",
		ProjectID: "project-other",
		ClusterID: "cluster-2",
	}).Error)
	application := entities.App{
		Base:           entities.Base{ID: "app-1"},
		Slug:           "app-1",
		Name:           "App 1",
		EnvID:          "other-env",
		ContainerImage: "registry.example.com/repo/app:old",
	}
	require.NoError(t, db.DB.Create(&application).Error)
	build := entities.Build{
		ID:             "build-1",
		BuildSettingID: setting.ID,
		BuildEnvID:     "build-env",
		BuildNumber:    1,
		Status:         entities.BuildStatusSucceeded,
		ImageFullName:  "registry.example.com/repo/app:new",
	}
	require.NoError(t, db.DB.Create(&build).Error)
	appID := application.ID
	deployment := entities.BuildDeployment{
		ID:         "deployment-1",
		BuildID:    build.ID,
		AppID:      &appID,
		EnvID:      "deploy-env",
		Status:     entities.BuildDeploymentStatusPending,
		DeployedBy: "auto",
	}
	require.NoError(t, db.DB.Create(&deployment).Error)

	handleCodeRepoBuildDeploy(&build, &deployment)

	var updatedDeployment entities.BuildDeployment
	require.NoError(t, db.DB.First(&updatedDeployment, "id = ?", deployment.ID).Error)
	assert.Equal(t, entities.BuildDeploymentStatusFailed, updatedDeployment.Status)
	assert.Equal(t, "deploy app must belong to the deploy environment", updatedDeployment.ErrorMessage)

	var updatedApp entities.App
	require.NoError(t, db.DB.First(&updatedApp, "id = ?", application.ID).Error)
	assert.Equal(t, "registry.example.com/repo/app:old", updatedApp.ContainerImage)
}

func TestUpdateBuildFailed_MarksPendingBuildDeploymentsFailed(t *testing.T) {
	setupBuildWatcherTestDB(t)

	now := time.Now()
	build := entities.Build{
		ID:             "build-1",
		BuildSettingID: "setting-1",
		BuildEnvID:     "env-1",
		BuildNumber:    1,
		Status:         entities.BuildStatusBuilding,
		StartedAt:      &now,
	}
	require.NoError(t, db.DB.Create(&build).Error)

	appID := "app-1"
	pendingDeployment := entities.BuildDeployment{
		ID:         "bd-pending",
		BuildID:    build.ID,
		AppID:      &appID,
		EnvID:      "env-1",
		Status:     entities.BuildDeploymentStatusPending,
		DeployedBy: "auto",
	}
	deployedDeployment := entities.BuildDeployment{
		ID:         "bd-deployed",
		BuildID:    build.ID,
		AppID:      &appID,
		EnvID:      "env-1",
		Status:     entities.BuildDeploymentStatusDeployed,
		DeployedBy: "auto",
	}
	require.NoError(t, db.DB.Create(&pendingDeployment).Error)
	require.NoError(t, db.DB.Create(&deployedDeployment).Error)

	updateBuildFailed(build.ID, "build job failed")

	var updatedBuild entities.Build
	require.NoError(t, db.DB.First(&updatedBuild, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildStatusFailed, updatedBuild.Status)
	assert.Equal(t, "build job failed", updatedBuild.ErrorMessage)
	require.NotNil(t, updatedBuild.CompletedAt)

	var updatedPending entities.BuildDeployment
	require.NoError(t, db.DB.First(&updatedPending, "id = ?", pendingDeployment.ID).Error)
	assert.Equal(t, entities.BuildDeploymentStatusFailed, updatedPending.Status)
	assert.Equal(t, "build job failed", updatedPending.ErrorMessage)
	require.NotNil(t, updatedPending.DeployedAt)

	var updatedDeployed entities.BuildDeployment
	require.NoError(t, db.DB.First(&updatedDeployed, "id = ?", deployedDeployment.ID).Error)
	assert.Equal(t, entities.BuildDeploymentStatusDeployed, updatedDeployed.Status)
	assert.Empty(t, updatedDeployed.ErrorMessage)
	assert.Nil(t, updatedDeployed.DeployedAt)
}

func TestRecoverActiveBuilds_FailsRowsMissingJobMetadata(t *testing.T) {
	setupBuildWatcherTestDB(t)

	env := entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "build-env",
		Name:             "Build Env",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "build-ns",
		IsBuildEnv:       true,
	}
	require.NoError(t, db.DB.Create(&env).Error)

	build := entities.Build{
		ID:             "build-missing-job",
		BuildSettingID: "setting-1",
		BuildEnvID:     env.ID,
		BuildNumber:    1,
		Status:         entities.BuildStatusPending,
	}
	require.NoError(t, db.DB.Create(&build).Error)

	bw := &BuildWatcher{watching: make(map[string]context.CancelFunc)}
	bw.RecoverActiveBuilds()

	require.Eventually(t, func() bool {
		var updated entities.Build
		if err := db.DB.First(&updated, "id = ?", build.ID).Error; err != nil {
			return false
		}
		return updated.Status == entities.BuildStatusFailed &&
			updated.ErrorMessage == "build job metadata missing after restart"
	}, time.Second, 10*time.Millisecond)

	bw.mu.Lock()
	defer bw.mu.Unlock()
	assert.Empty(t, bw.watching)
}

func TestWatchBuild_FailsAfterRepeatedJobNotFound(t *testing.T) {
	setupBuildWatcherTestDB(t)

	env := entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "build-env",
		Name:             "Build Env",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "build-ns",
		IsBuildEnv:       true,
	}
	require.NoError(t, db.DB.Create(&env).Error)

	now := time.Now()
	build := entities.Build{
		ID:             "build-job-missing",
		BuildSettingID: "setting-1",
		BuildEnvID:     env.ID,
		BuildNumber:    1,
		Status:         entities.BuildStatusPending,
		JobName:        "build-job-1",
		JobNamespace:   "build-ns",
		StartedAt:      &now,
	}
	require.NoError(t, db.DB.Create(&build).Error)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"kind":       "Status",
			"apiVersion": "v1",
			"status":     "Failure",
			"message":    "jobs.batch \"build-job-1\" not found",
			"reason":     "NotFound",
			"code":       http.StatusNotFound,
		}))
	}))
	defer server.Close()

	kubeConfig := "apiVersion: v1\n" +
		"kind: Config\n" +
		"clusters:\n" +
		"- name: test\n" +
		"  cluster:\n" +
		"    server: " + server.URL + "\n" +
		"contexts:\n" +
		"- name: test\n" +
		"  context:\n" +
		"    cluster: test\n" +
		"    user: test\n" +
		"current-context: test\n" +
		"users:\n" +
		"- name: test\n" +
		"  user: {}\n"

	require.NoError(t, kube.GlobalClusterStore.AddClient(env.ClusterID, kubeConfig))
	t.Cleanup(func() {
		kube.GlobalClusterStore.RemoveClient(env.ClusterID)
	})

	bw := &BuildWatcher{watching: make(map[string]context.CancelFunc)}
	ctx, cancel := context.WithTimeout(context.Background(), 17*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		bw.watchBuild(ctx, build.ID, env.ID, build.JobName, build.JobNamespace)
	}()
	<-done

	var updated entities.Build
	require.NoError(t, db.DB.First(&updated, "id = ?", build.ID).Error)
	assert.Equal(t, entities.BuildStatusFailed, updated.Status)
	assert.Equal(t, "build job not found after restart", updated.ErrorMessage)
	require.NotNil(t, updated.CompletedAt)
}

func TestBuildWatcherCancelsActiveWatchersOnParentContextCancel(t *testing.T) {
	t.Parallel()

	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	stopped := make(chan string, 2)
	bw := &BuildWatcher{
		watching:  make(map[string]context.CancelFunc),
		parentCtx: parentCtx,
		watchBuildFn: func(ctx context.Context, buildID, _, _, _ string) {
			<-ctx.Done()
			stopped <- buildID
		},
	}

	bw.StartWatching(&entities.Build{ID: "build-1", BuildEnvID: "env-1", JobName: "job-1", JobNamespace: "ns-1"})
	bw.StartWatching(&entities.Build{ID: "build-2", BuildEnvID: "env-1", JobName: "job-2", JobNamespace: "ns-1"})

	cancelParent()
	bw.Wait()

	assert.ElementsMatch(t, []string{"build-1", "build-2"}, []string{<-stopped, <-stopped})
}
