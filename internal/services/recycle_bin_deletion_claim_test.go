package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupRecycleBinDeletionClaimTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/recycle-bin-claim.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.User{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Env{},
		&entities.App{},
		&entities.AppGateway{},
		&entities.AppGatewayHTTPRoute{},
		&entities.AppGatewayHTTPRouteBackend{},
		&entities.AppEnvVar{},
		&entities.AppVolume{},
		&entities.AppConfigFile{},
		&entities.AppProbe{},
		&entities.AppPlugin{},
		&entities.AppAutoScaling{},
		&entities.AppSchedulingRule{},
		&entities.AppGroupMember{},
		&entities.AppFavorite{},
		&entities.DeploymentHistory{},
		&entities.BuildDeployment{},
		&entities.OperationLog{},
		&entities.RecycleBinDeletionClaim{},
	))

	db.DB = testDB
}

func seedRecycleBinDeletionHierarchy(t *testing.T, suffix, clusterID string) (string, string, string) {
	t.Helper()

	projectID := "project-" + suffix
	envID := "env-" + suffix
	appID := "app-" + suffix
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: projectID},
		Slug: projectID,
		Name: projectID,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: envID},
		Slug:             envID,
		Name:             envID,
		ProjectID:        projectID,
		ClusterID:        clusterID,
		ClusterNamespace: "work-" + suffix,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: appID},
		Slug:           appID,
		Name:           appID,
		EnvID:          envID,
		ContainerImage: "busybox",
	}).Error)
	return projectID, envID, appID
}

func seedRecycleBinDeletionClaim(t *testing.T, resourceType recycleBinResourceType, resourceID string) {
	t.Helper()
	require.NoError(t, db.DB.Create(&entities.RecycleBinDeletionClaim{
		ID:           "claim-" + string(resourceType) + "-" + resourceID,
		ResourceType: string(resourceType),
		ResourceID:   resourceID,
	}).Error)
}

func requireSoftDeleted(t *testing.T, model any, id string) {
	t.Helper()
	var count int64
	require.NoError(t, db.DB.Unscoped().Model(model).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestRestoreAppRejectsOwnAndAncestorDeletionClaims(t *testing.T) {
	tests := []struct {
		name          string
		claimType     recycleBinResourceType
		claimID       func(projectID, envID, appID string) string
		deleteEnv     bool
		deleteProject bool
	}{
		{
			name:      "application claim",
			claimType: recycleBinResourceApp,
			claimID:   func(_, _, appID string) string { return appID },
		},
		{
			name:      "environment claim",
			claimType: recycleBinResourceEnvironment,
			claimID:   func(_, envID, _ string) string { return envID },
			deleteEnv: true,
		},
		{
			name:          "project claim",
			claimType:     recycleBinResourceProject,
			claimID:       func(projectID, _, _ string) string { return projectID },
			deleteEnv:     true,
			deleteProject: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupRecycleBinDeletionClaimTestDB(t)
			projectID, envID, appID := seedRecycleBinDeletionHierarchy(t, tt.name, "cluster-unused")
			require.NoError(t, db.DB.Delete(&entities.App{}, "id = ?", appID).Error)
			if tt.deleteEnv {
				require.NoError(t, db.DB.Delete(&entities.Env{}, "id = ?", envID).Error)
			}
			if tt.deleteProject {
				require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", projectID).Error)
			}
			seedRecycleBinDeletionClaim(t, tt.claimType, tt.claimID(projectID, envID, appID))

			err := RestoreApp(t.Context(), appID)
			require.ErrorIs(t, err, ErrRecycleBinResourceDeleting)
			requireSoftDeleted(t, &entities.App{}, appID)
		})
	}
}

func TestRestoreEnvAndProjectRejectAncestorDeletionClaims(t *testing.T) {
	setupRecycleBinDeletionClaimTestDB(t)
	projectID, envID, _ := seedRecycleBinDeletionHierarchy(t, "ancestor", "cluster-unused")
	require.NoError(t, db.DB.Delete(&entities.Env{}, "id = ?", envID).Error)
	require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", projectID).Error)
	seedRecycleBinDeletionClaim(t, recycleBinResourceProject, projectID)

	require.ErrorIs(t, RestoreEnv(envID), ErrRecycleBinResourceDeleting)
	require.ErrorIs(t, RestoreProject(projectID), ErrRecycleBinResourceDeleting)
	requireSoftDeleted(t, &entities.Env{}, envID)
	requireSoftDeleted(t, &entities.Project{}, projectID)
}

func TestRestoreAppRejectsSoftDeletedParentsWithoutDeletionClaims(t *testing.T) {
	tests := []struct {
		name          string
		deleteEnv     bool
		deleteProject bool
	}{
		{name: "deleted environment", deleteEnv: true},
		{name: "deleted project", deleteProject: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupRecycleBinDeletionClaimTestDB(t)
			projectID, envID, appID := seedRecycleBinDeletionHierarchy(t, tt.name, "cluster-unused")
			require.NoError(t, db.DB.Delete(&entities.App{}, "id = ?", appID).Error)
			if tt.deleteEnv {
				require.NoError(t, db.DB.Delete(&entities.Env{}, "id = ?", envID).Error)
			}
			if tt.deleteProject {
				require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", projectID).Error)
			}

			err := RestoreApp(t.Context(), appID)
			require.ErrorIs(t, err, ErrRecycleBinParentDeleted)
			requireSoftDeleted(t, &entities.App{}, appID)
		})
	}
}

func TestRestoreEnvRejectsSoftDeletedProjectWithoutDeletionClaim(t *testing.T) {
	setupRecycleBinDeletionClaimTestDB(t)
	projectID, envID, _ := seedRecycleBinDeletionHierarchy(t, "deleted-parent", "cluster-unused")
	require.NoError(t, db.DB.Delete(&entities.Env{}, "id = ?", envID).Error)
	require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", projectID).Error)

	err := RestoreEnv(envID)
	require.ErrorIs(t, err, ErrRecycleBinParentDeleted)
	requireSoftDeleted(t, &entities.Env{}, envID)
}

func TestRestoreUserRejectsUserAndOwnedProjectDeletionClaims(t *testing.T) {
	setupRecycleBinDeletionClaimTestDB(t)
	userID := "user-claimed"
	projectID := "project-claimed"
	seedUserOwnedProject(t, userID, projectID)
	require.NoError(t, DeleteUser(userID))
	seedRecycleBinDeletionClaim(t, recycleBinResourceUser, userID)
	seedRecycleBinDeletionClaim(t, recycleBinResourceProject, projectID)

	require.ErrorIs(t, RestoreUser(userID), ErrRecycleBinResourceDeleting)
	require.ErrorIs(t, RestoreProject(projectID), ErrRecycleBinResourceDeleting)
	requireSoftDeleted(t, &entities.User{}, userID)
	requireSoftDeleted(t, &entities.Project{}, projectID)
}

func TestBatchRestoreRollsBackWhenOneProjectIsBeingDeleted(t *testing.T) {
	setupRecycleBinDeletionClaimTestDB(t)
	firstProjectID, _, _ := seedRecycleBinDeletionHierarchy(t, "batch-first", "cluster-unused")
	secondProjectID, _, _ := seedRecycleBinDeletionHierarchy(t, "batch-second", "cluster-unused")
	require.NoError(t, db.DB.Delete(&entities.Project{}, "id IN ?", []string{firstProjectID, secondProjectID}).Error)
	seedRecycleBinDeletionClaim(t, recycleBinResourceProject, secondProjectID)

	err := BatchRestoreProjects(
		[]string{firstProjectID, secondProjectID},
		RecycleBinActor{Role: app.UserRoleAdmin},
	)
	require.ErrorIs(t, err, ErrRecycleBinResourceDeleting)
	requireSoftDeleted(t, &entities.Project{}, firstProjectID)
	requireSoftDeleted(t, &entities.Project{}, secondProjectID)
}

func TestPermanentlyDeleteEnvBlocksRestoreDuringNamespaceCleanup(t *testing.T) {
	setupRecycleBinDeletionClaimTestDB(t)

	deleteStarted := make(chan struct{})
	releaseDelete := make(chan struct{})
	var signalOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete || request.URL.Path != "/api/v1/namespaces/work-race" {
			http.NotFound(w, request)
			return
		}
		signalOnce.Do(func() { close(deleteStarted) })
		<-releaseDelete
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	clusterID := registerAppVolumeTestCluster(t, server.URL)
	_, envID, appID := seedRecycleBinDeletionHierarchy(t, "race", clusterID)
	require.NoError(t, db.DB.Delete(&entities.App{}, "id = ?", appID).Error)
	require.NoError(t, db.DB.Delete(&entities.Env{}, "id = ?", envID).Error)

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- PermanentlyDeleteEnv(envID, RecycleBinActor{Role: app.UserRoleAdmin})
	}()

	select {
	case <-deleteStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("namespace deletion did not start")
	}

	restoreErr := RestoreEnv(envID)
	require.ErrorIs(t, restoreErr, ErrRecycleBinResourceDeleting)
	close(releaseDelete)

	select {
	case err := <-deleteResult:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("permanent deletion did not finish")
	}

	var envCount int64
	require.NoError(t, db.DB.Unscoped().Model(&entities.Env{}).Where("id = ?", envID).Count(&envCount).Error)
	require.Zero(t, envCount)
	var claimCount int64
	require.NoError(t, db.DB.Model(&entities.RecycleBinDeletionClaim{}).
		Where("resource_type = ? AND resource_id = ?", recycleBinResourceEnvironment, envID).
		Count(&claimCount).Error)
	require.Zero(t, claimCount)
}

func TestPermanentlyDeleteEnvRetainsClaimAfterCleanupFailureAndCanRetry(t *testing.T) {
	setupRecycleBinDeletionClaimTestDB(t)

	var cleanupSucceeds atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodDelete || request.URL.Path != "/api/v1/namespaces/work-retry" {
			http.NotFound(w, request)
			return
		}
		if !cleanupSucceeds.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			require.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{
				Status:  metav1.StatusFailure,
				Reason:  metav1.StatusReasonInternalError,
				Code:    http.StatusInternalServerError,
				Message: "temporary namespace cleanup failure",
			}))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	clusterID := registerAppVolumeTestCluster(t, server.URL)
	_, envID, appID := seedRecycleBinDeletionHierarchy(t, "retry", clusterID)
	require.NoError(t, db.DB.Delete(&entities.App{}, "id = ?", appID).Error)
	require.NoError(t, db.DB.Delete(&entities.Env{}, "id = ?", envID).Error)
	actor := RecycleBinActor{Role: app.UserRoleAdmin}

	require.Error(t, PermanentlyDeleteEnv(envID, actor))
	var claimCount int64
	require.NoError(t, db.DB.Model(&entities.RecycleBinDeletionClaim{}).
		Where("resource_type = ? AND resource_id = ?", recycleBinResourceEnvironment, envID).
		Count(&claimCount).Error)
	require.Equal(t, int64(1), claimCount)
	require.ErrorIs(t, RestoreEnv(envID), ErrRecycleBinResourceDeleting)

	cleanupSucceeds.Store(true)
	require.NoError(t, PermanentlyDeleteEnv(envID, actor))
	require.NoError(t, db.DB.Model(&entities.RecycleBinDeletionClaim{}).
		Where("resource_type = ? AND resource_id = ?", recycleBinResourceEnvironment, envID).
		Count(&claimCount).Error)
	require.Zero(t, claimCount)
	var envCount int64
	require.NoError(t, db.DB.Unscoped().Model(&entities.Env{}).Where("id = ?", envID).Count(&envCount).Error)
	require.Zero(t, envCount)
}

func TestPermanentDeletionRejectsActiveChildrenBeforeNamespaceCleanup(t *testing.T) {
	tests := []struct {
		name          string
		resourceType  recycleBinResourceType
		deleteEnv     bool
		deleteProject bool
		invoke        func(projectID, envID string, actor RecycleBinActor) error
	}{
		{
			name:         "single environment",
			resourceType: recycleBinResourceEnvironment,
			deleteEnv:    true,
			invoke: func(_ string, envID string, actor RecycleBinActor) error {
				return PermanentlyDeleteEnv(envID, actor)
			},
		},
		{
			name:         "batch environment",
			resourceType: recycleBinResourceEnvironment,
			deleteEnv:    true,
			invoke: func(_ string, envID string, actor RecycleBinActor) error {
				return BatchPermanentlyDeleteEnvs([]string{envID}, actor)
			},
		},
		{
			name:          "single project with active environment",
			resourceType:  recycleBinResourceProject,
			deleteProject: true,
			invoke: func(projectID, _ string, actor RecycleBinActor) error {
				return PermanentlyDeleteProject(projectID, actor)
			},
		},
		{
			name:          "batch project with active environment",
			resourceType:  recycleBinResourceProject,
			deleteProject: true,
			invoke: func(projectID, _ string, actor RecycleBinActor) error {
				return BatchPermanentlyDeleteProjects([]string{projectID}, actor)
			},
		},
		{
			name:          "project with active app under deleted environment",
			resourceType:  recycleBinResourceProject,
			deleteEnv:     true,
			deleteProject: true,
			invoke: func(projectID, _ string, actor RecycleBinActor) error {
				return PermanentlyDeleteProject(projectID, actor)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupRecycleBinDeletionClaimTestDB(t)

			var namespaceDeleteCount atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method == http.MethodDelete {
					namespaceDeleteCount.Add(1)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			clusterID := registerAppVolumeTestCluster(t, server.URL)
			projectID, envID, _ := seedRecycleBinDeletionHierarchy(t, "active-children-"+tt.name, clusterID)
			if tt.deleteEnv {
				require.NoError(t, db.DB.Delete(&entities.Env{}, "id = ?", envID).Error)
			}
			if tt.deleteProject {
				require.NoError(t, db.DB.Delete(&entities.Project{}, "id = ?", projectID).Error)
			}

			actor := RecycleBinActor{Role: app.UserRoleAdmin}
			err := tt.invoke(projectID, envID, actor)
			require.ErrorIs(t, err, ErrRecycleBinActiveChildren)
			require.Zero(t, namespaceDeleteCount.Load())

			resourceID := envID
			if tt.resourceType == recycleBinResourceProject {
				resourceID = projectID
			}
			var claimCount int64
			require.NoError(t, db.DB.Model(&entities.RecycleBinDeletionClaim{}).
				Where("resource_type = ? AND resource_id = ?", tt.resourceType, resourceID).
				Count(&claimCount).Error)
			require.Zero(t, claimCount, "failed preflight must roll back the deletion claim")
		})
	}
}
