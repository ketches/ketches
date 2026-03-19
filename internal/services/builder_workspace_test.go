package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func setupBuilderWorkspaceServiceTestDB(t *testing.T) {
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

func setBuilderWorkspaceServiceConfigForTest(t *testing.T) {
	t.Helper()

	originalConfig := app.Config
	app.Config = app.AppConfig{
		BuilderWorkspaceImage: "ghcr.io/ketches/builder-workspace:latest",
		BuilderWorkspaceRoot:  "/workspace",
	}
	t.Cleanup(func() {
		app.Config = originalConfig
	})
}

func seedBuilderWorkspaceServiceFixture(t *testing.T) (*entities.BuilderSession, *entities.BuilderWorkspace, *entities.BuilderRun) {
	t.Helper()

	now := time.Now().UTC()

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

	session := &entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-1",
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Status:         entities.BuilderSessionStatusProvisioning,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(session).Error)

	workspace := &entities.BuilderWorkspace{
		ID:            "workspace-1",
		CreatedAt:     now.Add(-4 * time.Minute),
		UpdatedAt:     now.Add(-4 * time.Minute),
		SessionID:     session.ID,
		BuildEnvID:    "env-1",
		ClusterID:     "cluster-1",
		Namespace:     "builder-ns",
		PodName:       "builder-workspace-session-1",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: "/workspace",
	}
	if err := db.DB.Create(workspace).Error; err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
		require.NoError(t, err)
	}

	runWorkspaceID := workspace.ID
	run := &entities.BuilderRun{
		ID:                 "run-1",
		CreatedAt:          now.Add(-3 * time.Minute),
		UpdatedAt:          now.Add(-3 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   "message-1",
		WorkspaceID:        &runWorkspaceID,
		Status:             entities.BuilderRunStatusExecuting,
		RequestedBy:        "user-1",
		InstructionSummary: "Create the initial project structure.",
	}
	require.NoError(t, db.DB.Create(run).Error)

	return session, workspace, run
}

func TestProvisionBuilderWorkspaceCreatesPodAndPVCRecords(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	session, _, _ := seedBuilderWorkspaceServiceFixture(t)

	require.NoError(t, db.DB.Where("session_id = ?", session.ID).Delete(&entities.BuilderWorkspace{}).Error)

	client := kubefake.NewSimpleClientset()
	originalGetBuilderWorkspaceClusterClient := getBuilderWorkspaceClusterClient
	t.Cleanup(func() {
		getBuilderWorkspaceClusterClient = originalGetBuilderWorkspaceClusterClient
	})
	getBuilderWorkspaceClusterClient = func(clusterID string) (kubernetes.Interface, error) {
		require.Equal(t, "cluster-1", clusterID)
		return client, nil
	}

	workspace, err := ProvisionBuilderWorkspace(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotNil(t, workspace)

	assert.Equal(t, session.ID, workspace.SessionID)
	assert.Equal(t, "env-1", workspace.BuildEnvID)
	assert.Equal(t, "cluster-1", workspace.ClusterID)
	assert.Equal(t, "builder-ns", workspace.Namespace)
	assert.Equal(t, app.Config.BuilderWorkspaceRoot, workspace.WorkspaceRoot)
	assert.Equal(t, entities.BuilderWorkspaceStatusActive, workspace.Status)

	stored := &entities.BuilderWorkspace{}
	require.NoError(t, db.DB.First(stored, "id = ?", workspace.ID).Error)
	assert.Equal(t, workspace.PodName, stored.PodName)
	assert.Equal(t, entities.BuilderWorkspaceStatusActive, stored.Status)

	updatedSession := &entities.BuilderSession{}
	require.NoError(t, db.DB.First(updatedSession, "id = ?", session.ID).Error)
	assert.Equal(t, entities.BuilderSessionStatusReady, updatedSession.Status)

	_, err = client.CoreV1().PersistentVolumeClaims("builder-ns").Get(context.Background(), workspace.PodName, metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().Pods("builder-ns").Get(context.Background(), workspace.PodName, metav1.GetOptions{})
	require.NoError(t, err)
}

func TestWriteAgentFilesRefreshesArtifactsFromWorkspaceRoot(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, run := seedBuilderWorkspaceServiceFixture(t)

	originalWriteBuilderWorkspaceFile := writeBuilderWorkspaceFile
	originalListBuilderWorkspaceFilesInContainer := listBuilderWorkspaceFilesInContainer
	t.Cleanup(func() {
		writeBuilderWorkspaceFile = originalWriteBuilderWorkspaceFile
		listBuilderWorkspaceFilesInContainer = originalListBuilderWorkspaceFilesInContainer
	})

	writtenPaths := make([]string, 0, 2)
	writeBuilderWorkspaceFile = func(_ *models.AppContext, podName, containerName, path, content string) error {
		assert.Equal(t, workspace.PodName, podName)
		assert.Equal(t, workspace.ContainerName, containerName)
		writtenPaths = append(writtenPaths, path+"="+content)
		return nil
	}
	listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ListFilesResponse, error) {
		assert.Equal(t, workspace.PodName, podName)
		assert.Equal(t, workspace.ContainerName, containerName)
		assert.Equal(t, workspace.WorkspaceRoot, path)
		return &models.ListFilesResponse{
			Path: workspace.WorkspaceRoot,
			Files: []models.FileInfo{
				{Name: "README.md", Type: "file", Size: 120},
				{Name: "package.json", Type: "file", Size: 80},
				{Name: "src", Type: "dir"},
			},
		}, nil
	}

	err := writeBuilderAgentFiles(context.Background(), workspace, run, []BuilderAgentFileWrite{
		{Path: "README.md", Content: "# Demo"},
		{Path: "package.json", Content: "{\"name\":\"demo\"}"},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"/workspace/README.md=# Demo",
		"/workspace/package.json={\"name\":\"demo\"}",
	}, writtenPaths)

	var artifacts []entities.BuilderArtifact
	require.NoError(t, db.DB.Where("session_id = ? AND workspace_id = ?", workspace.SessionID, workspace.ID).Order("path ASC").Find(&artifacts).Error)
	require.Len(t, artifacts, 2)
	assert.Equal(t, "README.md", artifacts[0].Path)
	assert.Equal(t, run.ID, artifacts[0].RunID)
	assert.Equal(t, "package.json", artifacts[1].Path)
	assert.Equal(t, run.ID, artifacts[1].RunID)
}

func TestDownloadBuilderWorkspaceStreamsWorkspaceRootContents(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, _ := seedBuilderWorkspaceServiceFixture(t)

	originalDownloadBuilderWorkspaceArchive := downloadBuilderWorkspaceArchive
	t.Cleanup(func() {
		downloadBuilderWorkspaceArchive = originalDownloadBuilderWorkspaceArchive
	})

	downloadBuilderWorkspaceArchive = func(_ *models.AppContext, podName, containerName, workspaceRoot string, writer io.Writer) error {
		assert.Equal(t, workspace.PodName, podName)
		assert.Equal(t, workspace.ContainerName, containerName)
		assert.Equal(t, workspace.WorkspaceRoot, workspaceRoot)
		_, err := writer.Write([]byte("archive-bytes"))
		return err
	}

	var buf bytes.Buffer
	err := DownloadBuilderWorkspace(context.Background(), "project-1", workspace.SessionID, &buf)
	require.NoError(t, err)
	assert.Equal(t, "archive-bytes", buf.String())
}

func TestListAndReadBuilderWorkspaceFilesRejectTraversalOutsideRoot(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, _ := seedBuilderWorkspaceServiceFixture(t)

	originalListBuilderWorkspaceFilesInContainer := listBuilderWorkspaceFilesInContainer
	originalReadBuilderWorkspaceFileInContainer := readBuilderWorkspaceFileInContainer
	t.Cleanup(func() {
		listBuilderWorkspaceFilesInContainer = originalListBuilderWorkspaceFilesInContainer
		readBuilderWorkspaceFileInContainer = originalReadBuilderWorkspaceFileInContainer
	})

	t.Run("list uses workspace root for safe relative path", func(t *testing.T) {
		listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ListFilesResponse, error) {
			assert.Equal(t, workspace.PodName, podName)
			assert.Equal(t, workspace.ContainerName, containerName)
			assert.Equal(t, "/workspace/src", path)
			return &models.ListFilesResponse{Path: path}, nil
		}

		result, err := ListBuilderWorkspaceFiles(context.Background(), "project-1", workspace.SessionID, "src")
		require.NoError(t, err)
		assert.Equal(t, "/workspace/src", result.Path)
	})

	t.Run("read uses workspace root for safe relative path", func(t *testing.T) {
		readBuilderWorkspaceFileInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ReadFileResponse, error) {
			assert.Equal(t, workspace.PodName, podName)
			assert.Equal(t, workspace.ContainerName, containerName)
			assert.Equal(t, "/workspace/README.md", path)
			return &models.ReadFileResponse{Path: path, Content: "# Demo", Size: 6}, nil
		}

		result, err := ReadBuilderWorkspaceFile(context.Background(), "project-1", workspace.SessionID, "README.md")
		require.NoError(t, err)
		assert.Equal(t, "/workspace/README.md", result.Path)
	})

	t.Run("list rejects traversal outside workspace root", func(t *testing.T) {
		called := false
		listBuilderWorkspaceFilesInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ListFilesResponse, error) {
			called = true
			return nil, nil
		}

		result, err := ListBuilderWorkspaceFiles(context.Background(), "project-1", workspace.SessionID, "../..")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.False(t, called)
	})

	t.Run("read rejects traversal outside workspace root", func(t *testing.T) {
		called := false
		readBuilderWorkspaceFileInContainer = func(_ *models.AppContext, podName, containerName, path string) (*models.ReadFileResponse, error) {
			called = true
			return nil, nil
		}

		result, err := ReadBuilderWorkspaceFile(context.Background(), "project-1", workspace.SessionID, "../../secret.txt")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.False(t, called)
	})
}
