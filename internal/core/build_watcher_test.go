package core

import (
	"strings"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
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

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.Build{},
		&entities.BuildDeployment{},
	))

	db.DB = testDB
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
