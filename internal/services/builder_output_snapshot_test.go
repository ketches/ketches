package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBuilderOutputSnapshotServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	db.DB = testDB
	require.NoError(t, db.Migrate())
}

func setupBuilderOutputSnapshotServiceFileTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	dbPath := filepath.Join(t.TempDir(), "builder-output-snapshot.db")
	testDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	sqlDB, err := testDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)

	db.DB = testDB
	require.NoError(t, db.Migrate())
}

func setBuilderOutputSnapshotServiceConfigForTest(t *testing.T) string {
	t.Helper()

	originalConfig := app.Config
	baseDir := t.TempDir()
	app.Config = app.AppConfig{
		BuilderSnapshotBaseDir: baseDir,
		BuilderWorkspaceRoot:   "/workspace",
	}
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	return baseDir
}

func seedBuilderOutputSnapshotFixture(t *testing.T) (*entities.BuilderSession, *entities.BuilderWorkspace, *entities.BuilderRun) {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	completedAt := now.Add(-time.Minute)

	project := &entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo-project",
		Name: "Demo Project",
	}
	require.NoError(t, db.DB.Create(project).Error)

	cluster := &entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-1",
		Name:       "Cluster 1",
		KubeConfig: "apiVersion: v1",
	}
	require.NoError(t, db.DB.Create(cluster).Error)

	env := &entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "build-env",
		Name:             "Build Env",
		ProjectID:        project.ID,
		ClusterID:        cluster.ID,
		ClusterNamespace: "builder-ns",
		IsBuildEnv:       true,
	}
	require.NoError(t, db.DB.Create(env).Error)

	session := &entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-1",
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
		},
		ProjectID:      project.ID,
		BuildEnvID:     env.ID,
		Title:          "Publish snapshot",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(session).Error)

	workspace := &entities.BuilderWorkspace{
		ID:            "workspace-1",
		CreatedAt:     now.Add(-4 * time.Minute),
		UpdatedAt:     now.Add(-4 * time.Minute),
		SessionID:     session.ID,
		BuildEnvID:    env.ID,
		ClusterID:     cluster.ID,
		Namespace:     env.ClusterNamespace,
		PodName:       "builder-workspace-session-1",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: "/workspace",
	}
	require.NoError(t, db.DB.Create(workspace).Error)

	run := &entities.BuilderRun{
		ID:               "run-1",
		CreatedAt:        now.Add(-3 * time.Minute),
		UpdatedAt:        now.Add(-2 * time.Minute),
		SessionID:        session.ID,
		TriggerMessageID: "message-1",
		WorkspaceID:      builderStringPtr(workspace.ID),
		Status:           entities.BuilderRunStatusSucceeded,
		RequestedBy:      "user-1",
		CompletedAt:      &completedAt,
	}
	require.NoError(t, db.DB.Create(run).Error)

	return session, workspace, run
}

func seedBuilderOutputArtifacts(t *testing.T, session *entities.BuilderSession, workspace *entities.BuilderWorkspace, run *entities.BuilderRun, files map[string]string) {
	t.Helper()

	now := time.Now().UTC()
	artifacts := make([]entities.BuilderArtifact, 0, len(files))
	for relativePath, content := range files {
		outputRoot := strings.SplitN(relativePath, "/", 2)[0]
		metadataJSON, err := marshalBuilderArtifactMetadata(int64(len(content)), outputRoot)
		require.NoError(t, err)

		artifacts = append(artifacts, entities.BuilderArtifact{
			ID:           "artifact-" + run.ID + "-" + strings.ReplaceAll(relativePath, "/", "-"),
			CreatedAt:    now,
			UpdatedAt:    now,
			SessionID:    session.ID,
			WorkspaceID:  workspace.ID,
			RunID:        run.ID,
			Kind:         entities.BuilderArtifactKindBuildOutput,
			Path:         relativePath,
			MetadataJSON: metadataJSON,
		})
	}

	require.NoError(t, db.DB.Create(&artifacts).Error)
}

func setBuilderOutputSnapshotSourceForTest(t *testing.T, sourceRoot string) {
	t.Helper()

	originalStreamFn := streamBuilderOutputSnapshotSourceFile
	streamBuilderOutputSnapshotSourceFile = func(_ context.Context, _ *entities.BuilderWorkspace, relativePath string, dst io.Writer) error {
		sourcePath := filepath.Join(sourceRoot, filepath.FromSlash(relativePath))
		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			return err
		}
		defer func() {
			require.NoError(t, sourceFile.Close())
		}()

		_, err = io.Copy(dst, sourceFile)
		return err
	}
	t.Cleanup(func() {
		streamBuilderOutputSnapshotSourceFile = originalStreamFn
	})
}

func writeBuilderOutputSnapshotSourceFiles(t *testing.T, sourceRoot string, files map[string]string) {
	t.Helper()

	for relativePath, content := range files {
		absPath := filepath.Join(sourceRoot, filepath.FromSlash(relativePath))
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
		require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))
	}
}

func TestPublishBuilderOutputSnapshot_CopiesBuildOutputIntoDurableStorage(t *testing.T) {
	setupBuilderOutputSnapshotServiceTestDB(t)
	baseDir := setBuilderOutputSnapshotServiceConfigForTest(t)
	session, workspace, run := seedBuilderOutputSnapshotFixture(t)

	sourceRoot := t.TempDir()
	writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
		"dist/index.html":    "<html><body>preview</body></html>",
		"dist/assets/app.js": "console.log('preview');\n",
	})
	seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
		"dist/index.html":    "<html><body>preview</body></html>",
		"dist/assets/app.js": "console.log('preview');\n",
	})
	setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

	snapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, run)
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	assert.Equal(t, session.ID, snapshot.SessionID)
	assert.Equal(t, run.ID, snapshot.RunID)
	assert.Equal(t, entities.BuilderOutputSnapshotStatusPreviewable, snapshot.Status)
	assert.Equal(t, "dist", snapshot.OutputRoot)
	assert.Equal(t, "dist/index.html", snapshot.DefaultEntryPath)
	assert.Equal(t, 2, snapshot.FileCount)
	assert.Equal(t, int64(len("<html><body>preview</body></html>")+len("console.log('preview');\n")), snapshot.TotalSizeBytes)

	latestSnapshot, err := GetLatestSuccessfulBuilderOutputSnapshot(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSnapshot)
	assert.Equal(t, snapshot.ID, latestSnapshot.ID)

	var storedFiles []entities.BuilderOutputSnapshotFile
	require.NoError(t, db.DB.Where("snapshot_id = ?", snapshot.ID).Order("relative_path ASC").Find(&storedFiles).Error)
	require.Len(t, storedFiles, 2)

	filesByPath := make(map[string]entities.BuilderOutputSnapshotFile, len(storedFiles))
	for _, file := range storedFiles {
		filesByPath[file.RelativePath] = file
		assert.FileExists(t, filepath.Join(baseDir, filepath.FromSlash(file.StoragePath)))
	}

	indexFile := filesByPath["dist/index.html"]
	require.True(t, indexFile.IsDefaultEntry)
	assert.Contains(t, indexFile.ContentType, "text/html")

	reader, err := OpenBuilderOutputSnapshotFile(&indexFile)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, reader.Close())
	}()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "<html><body>preview</body></html>", string(content))

	assetFile := filesByPath["dist/assets/app.js"]
	assert.False(t, assetFile.IsDefaultEntry)
	assert.Contains(t, assetFile.ContentType, "javascript")
	assert.Contains(t, assetFile.StoragePath, snapshot.StoragePath)
	assert.FileExists(t, filepath.Join(baseDir, filepath.FromSlash(snapshot.StoragePath), "dist", "assets", "app.js"))
}

func TestPublishBuilderOutputSnapshot_MarksDeliveryOnlyWhenIndexHTMLMissing(t *testing.T) {
	setupBuilderOutputSnapshotServiceTestDB(t)
	setBuilderOutputSnapshotServiceConfigForTest(t)
	session, workspace, run := seedBuilderOutputSnapshotFixture(t)

	sourceRoot := t.TempDir()
	writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
		"dist/assets/app.js": "console.log('delivery');\n",
		"dist/style.css":     "body { color: black; }\n",
	})
	seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
		"dist/assets/app.js": "console.log('delivery');\n",
		"dist/style.css":     "body { color: black; }\n",
	})
	setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

	snapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, run)
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	assert.Equal(t, entities.BuilderOutputSnapshotStatusDeliveryOnly, snapshot.Status)
	assert.Empty(t, snapshot.DefaultEntryPath)
	assert.Equal(t, "dist", snapshot.OutputRoot)

	var entryCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshotFile{}).Where("snapshot_id = ? AND is_default_entry = ?", snapshot.ID, true).Count(&entryCount).Error)
	assert.Equal(t, int64(0), entryCount)

	latestSnapshot, err := GetLatestSuccessfulBuilderOutputSnapshot(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSnapshot)
	assert.Equal(t, snapshot.ID, latestSnapshot.ID)
	assert.Equal(t, entities.BuilderOutputSnapshotStatusDeliveryOnly, latestSnapshot.Status)
}

func TestPublishBuilderOutputSnapshot_MarksDeliveryOnlyForNextOutputRoot(t *testing.T) {
	setupBuilderOutputSnapshotServiceTestDB(t)
	setBuilderOutputSnapshotServiceConfigForTest(t)
	session, workspace, run := seedBuilderOutputSnapshotFixture(t)

	sourceRoot := t.TempDir()
	writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
		".next/routes-manifest.json": "{\"version\":1}\n",
		".next/static/app.js":        "console.log('ssr');\n",
	})
	seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
		".next/routes-manifest.json": "{\"version\":1}\n",
		".next/static/app.js":        "console.log('ssr');\n",
	})
	setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

	snapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, run)
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	assert.Equal(t, entities.BuilderOutputSnapshotStatusDeliveryOnly, snapshot.Status)
	assert.Empty(t, snapshot.DefaultEntryPath)
	assert.Equal(t, ".next", snapshot.OutputRoot)

	latestSnapshot, err := GetLatestSuccessfulBuilderOutputSnapshot(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSnapshot)
	assert.Equal(t, snapshot.ID, latestSnapshot.ID)
	assert.Equal(t, entities.BuilderOutputSnapshotStatusDeliveryOnly, latestSnapshot.Status)
}

func TestPublishBuilderOutputSnapshot_RejectsNonSucceededRuns(t *testing.T) {
	statuses := []entities.BuilderRunStatus{
		entities.BuilderRunStatusExecuting,
		entities.BuilderRunStatusFailed,
		entities.BuilderRunStatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			setupBuilderOutputSnapshotServiceTestDB(t)
			baseDir := setBuilderOutputSnapshotServiceConfigForTest(t)
			session, workspace, run := seedBuilderOutputSnapshotFixture(t)
			run.Status = status
			require.NoError(t, db.DB.Save(run).Error)

			sourceRoot := t.TempDir()
			writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
				"dist/index.html":    "<html><body>preview</body></html>",
				"dist/assets/app.js": "console.log('preview');\n",
			})
			seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
				"dist/index.html":    "<html><body>preview</body></html>",
				"dist/assets/app.js": "console.log('preview');\n",
			})
			setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

			snapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, run)
			require.Error(t, err)
			assert.Nil(t, snapshot)
			assert.Contains(t, err.Error(), "builder run must be succeeded")

			var snapshotCount int64
			require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshot{}).Count(&snapshotCount).Error)
			assert.Equal(t, int64(0), snapshotCount)

			var snapshotFileCount int64
			require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshotFile{}).Count(&snapshotFileCount).Error)
			assert.Equal(t, int64(0), snapshotFileCount)

			entries, err := os.ReadDir(baseDir)
			require.NoError(t, err)
			assert.Empty(t, entries)
		})
	}
}

func TestPublishBuilderOutputSnapshot_RejectsMismatchedWorkspaceAndRunLineage(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*entities.BuilderWorkspace, *entities.BuilderRun) *entities.BuilderWorkspace
		errContains string
	}{
		{
			name: "session mismatch",
			mutate: func(workspace *entities.BuilderWorkspace, _ *entities.BuilderRun) *entities.BuilderWorkspace {
				mismatched := *workspace
				mismatched.ID = "workspace-session-mismatch"
				mismatched.SessionID = "session-other"
				return &mismatched
			},
			errContains: "builder workspace session id must match builder run session id",
		},
		{
			name: "workspace mismatch",
			mutate: func(workspace *entities.BuilderWorkspace, _ *entities.BuilderRun) *entities.BuilderWorkspace {
				mismatched := *workspace
				mismatched.ID = "workspace-other"
				return &mismatched
			},
			errContains: "builder workspace id must match builder run workspace id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupBuilderOutputSnapshotServiceTestDB(t)
			baseDir := setBuilderOutputSnapshotServiceConfigForTest(t)
			session, workspace, run := seedBuilderOutputSnapshotFixture(t)

			sourceRoot := t.TempDir()
			writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
				"dist/index.html": "<html><body>preview</body></html>",
			})
			seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
				"dist/index.html": "<html><body>preview</body></html>",
			})
			setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

			mismatchedWorkspace := tc.mutate(workspace, run)
			snapshot, err := PublishBuilderOutputSnapshot(context.Background(), mismatchedWorkspace, run)
			require.Error(t, err)
			assert.Nil(t, snapshot)
			assert.Contains(t, err.Error(), tc.errContains)

			var snapshotCount int64
			require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshot{}).Count(&snapshotCount).Error)
			assert.Equal(t, int64(0), snapshotCount)

			entries, err := os.ReadDir(baseDir)
			require.NoError(t, err)
			assert.Empty(t, entries)
		})
	}
}

func TestGetLatestSuccessfulBuilderOutputSnapshot_PrefersLatestSuccessfulRunOverLaterPublication(t *testing.T) {
	setupBuilderOutputSnapshotServiceTestDB(t)
	setBuilderOutputSnapshotServiceConfigForTest(t)
	session, workspace, olderRun := seedBuilderOutputSnapshotFixture(t)

	newerCompletedAt := olderRun.CreatedAt.Add(2 * time.Minute)
	newerRun := &entities.BuilderRun{
		ID:               "run-2",
		CreatedAt:        olderRun.CreatedAt.Add(time.Minute),
		UpdatedAt:        olderRun.CreatedAt.Add(time.Minute),
		SessionID:        session.ID,
		TriggerMessageID: "message-2",
		WorkspaceID:      builderStringPtr(workspace.ID),
		Status:           entities.BuilderRunStatusSucceeded,
		RequestedBy:      "user-1",
		CompletedAt:      &newerCompletedAt,
	}
	require.NoError(t, db.DB.Create(newerRun).Error)

	sourceRoot := t.TempDir()
	setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

	writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
		"dist/index.html": "<html><body>newer</body></html>",
	})
	seedBuilderOutputArtifacts(t, session, workspace, newerRun, map[string]string{
		"dist/index.html": "<html><body>newer</body></html>",
	})
	newerSnapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, newerRun)
	require.NoError(t, err)
	require.NotNil(t, newerSnapshot)

	writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
		"dist/index.html": "<html><body>older</body></html>",
	})
	seedBuilderOutputArtifacts(t, session, workspace, olderRun, map[string]string{
		"dist/index.html": "<html><body>older</body></html>",
	})
	olderSnapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, olderRun)
	require.NoError(t, err)
	require.NotNil(t, olderSnapshot)

	latestSnapshot, err := GetLatestSuccessfulBuilderOutputSnapshot(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSnapshot)
	assert.Equal(t, newerRun.ID, latestSnapshot.RunID)
	assert.Equal(t, newerSnapshot.ID, latestSnapshot.ID)
}

func TestPublishBuilderOutputSnapshot_CleansUpFinalizedStorageWhenDatabaseWriteFails(t *testing.T) {
	setupBuilderOutputSnapshotServiceTestDB(t)
	baseDir := setBuilderOutputSnapshotServiceConfigForTest(t)
	session, workspace, run := seedBuilderOutputSnapshotFixture(t)

	sourceRoot := t.TempDir()
	writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
		"dist/index.html": "<html><body>preview</body></html>",
	})
	seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
		"dist/index.html": "<html><body>preview</body></html>",
	})
	setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

	callbackName := "test:fail-builder-output-snapshot-file-create"
	forcedErr := errors.New("forced snapshot file create failure")
	require.NoError(t, db.DB.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Schema == nil || tx.Statement.Schema.Table != "builder_output_snapshot_files" {
			return
		}
		_ = tx.AddError(forcedErr)
	}))
	t.Cleanup(func() {
		require.NoError(t, db.DB.Callback().Create().Remove(callbackName))
	})

	snapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, run)
	require.ErrorIs(t, err, forcedErr)
	assert.Nil(t, snapshot)

	var snapshotCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshot{}).Count(&snapshotCount).Error)
	assert.Equal(t, int64(0), snapshotCount)

	var snapshotFileCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshotFile{}).Count(&snapshotFileCount).Error)
	assert.Equal(t, int64(0), snapshotFileCount)

	runStorageRoot := filepath.Join(baseDir, "sessions", session.ID, "runs", run.ID)
	entries, readErr := os.ReadDir(runStorageRoot)
	if errors.Is(readErr, os.ErrNotExist) {
		return
	}
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

func TestPublishBuilderOutputSnapshot_ReturnsExistingSnapshotOnCompetingPublication(t *testing.T) {
	setupBuilderOutputSnapshotServiceFileTestDB(t)
	baseDir := setBuilderOutputSnapshotServiceConfigForTest(t)
	session, workspace, run := seedBuilderOutputSnapshotFixture(t)

	sourceRoot := t.TempDir()
	writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
		"dist/index.html": "<html><body>preview</body></html>",
	})
	seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
		"dist/index.html": "<html><body>preview</body></html>",
	})

	originalStreamFn := streamBuilderOutputSnapshotSourceFile
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	streamBuilderOutputSnapshotSourceFile = func(_ context.Context, _ *entities.BuilderWorkspace, relativePath string, dst io.Writer) error {
		arrived <- struct{}{}
		<-release

		sourcePath := filepath.Join(sourceRoot, filepath.FromSlash(relativePath))
		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			return err
		}
		defer func() {
			require.NoError(t, sourceFile.Close())
		}()

		_, err = io.Copy(dst, sourceFile)
		return err
	}
	t.Cleanup(func() {
		streamBuilderOutputSnapshotSourceFile = originalStreamFn
	})

	var wg sync.WaitGroup
	wg.Add(2)
	results := make([]*entities.BuilderOutputSnapshot, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = PublishBuilderOutputSnapshot(context.Background(), workspace, run)
		}(i)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for competing publication stream %d", i+1)
		}
	}
	close(release)
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.NotNil(t, results[0])
	require.NotNil(t, results[1])
	assert.Equal(t, results[0].ID, results[1].ID)
	assert.Equal(t, run.ID, results[0].RunID)
	assert.Equal(t, run.ID, results[1].RunID)

	var snapshotCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshot{}).Count(&snapshotCount).Error)
	assert.Equal(t, int64(1), snapshotCount)

	runStorageRoot := filepath.Join(baseDir, "sessions", session.ID, "runs", run.ID)
	entries, err := os.ReadDir(runStorageRoot)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestPublishBuilderOutputSnapshot_RejectsBuildOutputPathLongerThanIndexedLimit(t *testing.T) {
	setupBuilderOutputSnapshotServiceTestDB(t)
	setBuilderOutputSnapshotServiceConfigForTest(t)
	session, workspace, run := seedBuilderOutputSnapshotFixture(t)

	tooLongPath := "dist/" + strings.Repeat("a", 251)
	seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
		tooLongPath: "console.log('preview');\n",
	})

	snapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, run)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBuilderAgentUnsafeFilePath)
	assert.Nil(t, snapshot)

	var snapshotCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshot{}).Count(&snapshotCount).Error)
	assert.Equal(t, int64(0), snapshotCount)
}

func TestWriteBuilderOutputSnapshotArchive_StreamsPublishedSnapshotAsTarGz(t *testing.T) {
	setupBuilderOutputSnapshotServiceTestDB(t)
	setBuilderOutputSnapshotServiceConfigForTest(t)
	session, workspace, run := seedBuilderOutputSnapshotFixture(t)

	sourceRoot := t.TempDir()
	writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
		"dist/index.html":    "<html><body>preview</body></html>",
		"dist/assets/app.js": "console.log('preview');\n",
	})
	seedBuilderOutputArtifacts(t, session, workspace, run, map[string]string{
		"dist/index.html":    "<html><body>preview</body></html>",
		"dist/assets/app.js": "console.log('preview');\n",
	})
	setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

	snapshot, err := PublishBuilderOutputSnapshot(context.Background(), workspace, run)
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	var archive bytes.Buffer
	err = WriteBuilderOutputSnapshotArchive(context.Background(), snapshot, &archive)
	require.NoError(t, err)
	require.NotZero(t, archive.Len())

	gzipReader, err := gzip.NewReader(bytes.NewReader(archive.Bytes()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, gzipReader.Close())
	}()

	tarReader := tar.NewReader(gzipReader)
	entries := map[string]string{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		data, err := io.ReadAll(tarReader)
		require.NoError(t, err)
		entries[header.Name] = string(data)
	}

	assert.Equal(t, map[string]string{
		"dist/index.html":    "<html><body>preview</body></html>",
		"dist/assets/app.js": "console.log('preview');\n",
	}, entries)
}
