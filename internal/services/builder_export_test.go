package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBuilderSessionExport(t *testing.T) {
	t.Run("creates export from latest successful snapshot when available", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)

		now := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-export-snapshot"},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Export snapshot session",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: now,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
			ID:               "snapshot-export-1",
			SessionID:        "session-export-snapshot",
			RunID:            "run-export-1",
			WorkspaceID:      "workspace-export-1",
			Status:           entities.BuilderOutputSnapshotStatusPreviewable,
			OutputRoot:       "dist",
			DefaultEntryPath: "dist/index.html",
			StoragePath:      "sessions/export/run-1/snapshot",
			FileCount:        2,
			TotalSizeBytes:   1024,
			PublishedAt:      now,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderRun{
			ID:                 "run-export-1",
			SessionID:          "session-export-snapshot",
			TriggerMessageID:   "message-export-1",
			Status:             entities.BuilderRunStatusSucceeded,
			RequestedBy:        "user-1",
			InstructionSummary: "Build app",
			CompletedAt:        &now,
		}).Error)

		resp, err := CreateBuilderSessionExport(context.Background(), "project-1", "session-export-snapshot", "user-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "session-export-snapshot", resp.SessionID)
		assert.Equal(t, "run-export-1", resp.RunID)
		assert.Equal(t, "snapshot-export-1", resp.SnapshotID)
		assert.Equal(t, "session_archive", resp.Kind)
		assert.Equal(t, "ready", resp.Status)
		assert.Contains(t, resp.FileName, "builder-output-run-export-1.tar.gz")
	})

	t.Run("falls back to workspace export when no snapshot exists", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)

		now := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, db.DB.Create(&entities.Project{
			Base: entities.Base{ID: "project-1"},
			Slug: "demo",
			Name: "Demo Project",
		}).Error)
		require.NoError(t, db.DB.Create(&entities.Cluster{
			Base:       entities.Base{ID: "cluster-1"},
			Slug:       "cluster-1",
			Name:       "Cluster 1",
			KubeConfig: "apiVersion: v1",
		}).Error)
		require.NoError(t, db.DB.Create(&entities.Env{
			Base:             entities.Base{ID: "env-1"},
			Slug:             "build-env",
			Name:             "Build Env",
			ProjectID:        "project-1",
			ClusterID:        "cluster-1",
			ClusterNamespace: "builder-ns",
			IsBuildEnv:       true,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-export-workspace"},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Export workspace session",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: now,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderWorkspace{
			ID:            "workspace-export-fallback",
			SessionID:     "session-export-workspace",
			BuildEnvID:    "env-1",
			ClusterID:     "cluster-1",
			Namespace:     "builder-ns",
			PodName:       "builder-workspace-session-export-workspace",
			ContainerName: "workspace",
			Status:        entities.BuilderWorkspaceStatusActive,
			WorkspaceRoot: "/workspace",
		}).Error)

		resp, err := CreateBuilderSessionExport(context.Background(), "project-1", "session-export-workspace", "user-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "session-export-workspace", resp.SessionID)
		assert.Equal(t, "workspace-export-fallback", resp.WorkspaceID)
		assert.Empty(t, resp.SnapshotID)
		assert.Contains(t, resp.FileName, "builder-workspace-session-export-workspace.tar.gz")
	})
}

func TestListAndDownloadBuilderSessionExport(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo",
		Name: "Demo Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-1",
		Name:       "Cluster 1",
		KubeConfig: "apiVersion: v1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "build-env",
		Name:             "Build Env",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "builder-ns",
		IsBuildEnv:       true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base:           entities.Base{ID: "session-export-list"},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Export list session",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now,
	}).Error)

	snapshotID := "snapshot-export-list"
	require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
		ID:               snapshotID,
		SessionID:        "session-export-list",
		RunID:            "run-export-list",
		WorkspaceID:      "workspace-export-list",
		Status:           entities.BuilderOutputSnapshotStatusPreviewable,
		OutputRoot:       "dist",
		DefaultEntryPath: "dist/index.html",
		StoragePath:      "sessions/export/list/snapshot",
		FileCount:        1,
		TotalSizeBytes:   512,
		PublishedAt:      now,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-export-list",
		SessionID:          "session-export-list",
		TriggerMessageID:   "message-export-list",
		Status:             entities.BuilderRunStatusSucceeded,
		RequestedBy:        "user-1",
		InstructionSummary: "Build app",
		CompletedAt:        &now,
	}).Error)

	exportResp, err := CreateBuilderSessionExport(context.Background(), "project-1", "session-export-list", "user-1")
	require.NoError(t, err)
	require.NotNil(t, exportResp)

	exports, err := ListBuilderSessionExports(context.Background(), "project-1", "session-export-list")
	require.NoError(t, err)
	require.Len(t, exports, 1)
	assert.Equal(t, exportResp.ID, exports[0].ID)

	originalWriteSnapshotArchive := writeBuilderExportSnapshotArchive
	t.Cleanup(func() {
		writeBuilderExportSnapshotArchive = originalWriteSnapshotArchive
	})
	writeBuilderExportSnapshotArchive = func(ctx context.Context, snapshot *entities.BuilderOutputSnapshot, writer io.Writer) error {
		_, err := writer.Write([]byte("export-snapshot-archive"))
		return err
	}

	var archive bytes.Buffer
	err = DownloadBuilderSessionExport(context.Background(), "project-1", "session-export-list", exportResp.ID, &archive)
	require.NoError(t, err)
	assert.Equal(t, "export-snapshot-archive", archive.String())
}

func TestPromoteBuilderSessionExportToCodeRepository(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo",
		Name: "Demo Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderSession{
		Base:           entities.Base{ID: "session-export-promote"},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Export promote session",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderExport{
		ID:          "export-promote-1",
		SessionID:   "session-export-promote",
		Kind:        "session_archive",
		Status:      entities.BuilderExportStatusReady,
		FileName:    "builder-output-run-promote-1.tar.gz",
		StoragePath: "builder-exports/session-export-promote/builder-output-run-promote-1.tar.gz",
		CreatedBy:   "user-1",
	}).Error)

	bareRepoDir := filepath.Join(t.TempDir(), "repo.git")
	cmd := exec.Command("git", "init", "--bare", bareRepoDir)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	originalDownloadBuilderExportArchive := downloadBuilderExportArchive
	t.Cleanup(func() {
		downloadBuilderExportArchive = originalDownloadBuilderExportArchive
	})
	downloadBuilderExportArchive = func(ctx context.Context, projectID, sessionID, exportID string, writer io.Writer) error {
		return writeTestTarGz(map[string]string{
			"README.md": "# Builder Export\n",
		}, writer)
	}

	resp, err := PromoteBuilderSessionExportToCodeRepository(context.Background(), "project-1", "session-export-promote", "export-promote-1", &models.PromoteBuilderExportToCodeRepositoryRequest{
		Name:       "Builder Export Repo",
		Slug:       "builder-export-repo",
		GitRepoURL: bareRepoDir,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "builder-export-repo", resp.Repository.Slug)

	showCmd := exec.Command("git", "--git-dir", bareRepoDir, "show", "main:README.md")
	showOutput, err := showCmd.CombinedOutput()
	require.NoError(t, err, string(showOutput))
	assert.Contains(t, string(showOutput), "Builder Export")

	var export entities.BuilderExport
	require.NoError(t, db.DB.First(&export, "id = ?", "export-promote-1").Error)
	assert.Contains(t, export.MetadataJSON, `"promoted_code_repository_id"`)
}

func TestGetBuilderSessionExportPromotionPlan(t *testing.T) {
	t.Run("allows initial build when workspace source export has Dockerfile", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)

		now := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-export-plan-workspace"},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Go API Service",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: now,
		}).Error)
		runID := "run-export-plan-workspace"
		require.NoError(t, db.DB.Create(&entities.BuilderRun{
			ID:                 runID,
			SessionID:          "session-export-plan-workspace",
			TriggerMessageID:   "message-export-plan-workspace",
			Status:             entities.BuilderRunStatusSucceeded,
			RequestedBy:        "user-1",
			InstructionSummary: "Build app",
			PlannedProjectKind: builderStringPtr("go_api_service"),
			CompletedAt:        &now,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderExport{
			ID:          "export-plan-workspace",
			SessionID:   "session-export-plan-workspace",
			RunID:       &runID,
			WorkspaceID: builderStringPtr("workspace-1"),
			Kind:        "session_archive",
			Status:      entities.BuilderExportStatusReady,
			FileName:    "builder-workspace.tar.gz",
			StoragePath: "builder-exports/session-export-plan-workspace/builder-workspace.tar.gz",
			CreatedBy:   "user-1",
		}).Error)
		require.NoError(t, db.DB.Create(&entities.BuilderArtifact{
			ID:           "artifact-dockerfile",
			SessionID:    "session-export-plan-workspace",
			WorkspaceID:  "workspace-1",
			RunID:        runID,
			Kind:         entities.BuilderArtifactKindWorkspaceFile,
			Path:         "Dockerfile",
			MetadataJSON: `{"size_bytes":42}`,
		}).Error)

		resp, err := GetBuilderSessionExportPromotionPlan(context.Background(), "project-1", "session-export-plan-workspace", "export-plan-workspace")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "workspace_source", resp.SourceKind)
		assert.Equal(t, "go_api_service", resp.PlannedProjectKind)
		assert.True(t, resp.CanTriggerInitialBuild)
		assert.Equal(t, "go-api-service", resp.SuggestedRepositorySlug)
		assert.Equal(t, "go-api-service", resp.SuggestedImageName)
	})

	t.Run("blocks initial build for snapshot-only export", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)

		now := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, db.DB.Create(&entities.BuilderSession{
			Base:           entities.Base{ID: "session-export-plan-snapshot"},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Static Site",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-1",
			LastActivityAt: now,
		}).Error)
		snapshotID := "snapshot-plan-snapshot"
		require.NoError(t, db.DB.Create(&entities.BuilderExport{
			ID:          "export-plan-snapshot",
			SessionID:   "session-export-plan-snapshot",
			SnapshotID:  &snapshotID,
			Kind:        "session_archive",
			Status:      entities.BuilderExportStatusReady,
			FileName:    "builder-output-run-1.tar.gz",
			StoragePath: "builder-exports/session-export-plan-snapshot/builder-output-run-1.tar.gz",
			CreatedBy:   "user-1",
		}).Error)

		resp, err := GetBuilderSessionExportPromotionPlan(context.Background(), "project-1", "session-export-plan-snapshot", "export-plan-snapshot")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "snapshot_output", resp.SourceKind)
		assert.False(t, resp.CanTriggerInitialBuild)
		assert.Contains(t, resp.MissingRequirements[0], "workspace source export")
	})
}

func writeTestTarGz(files map[string]string, writer io.Writer) error {
	gzipWriter := gzip.NewWriter(writer)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}

	if err := tarWriter.Close(); err != nil {
		return err
	}
	return gzipWriter.Close()
}
