package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const builderWorkspaceStorageRequest = "5Gi"

var getBuilderWorkspaceClusterClient = func(clusterID string) (kubernetes.Interface, error) {
	return kube.GlobalClusterStore.GetClient(clusterID)
}

var writeBuilderWorkspaceFile = func(appCtx *models.AppContext, podName, containerName, filePath, content string) error {
	return WriteFile(appCtx, podName, containerName, filePath, content)
}

var listBuilderWorkspaceFilesInContainer = func(appCtx *models.AppContext, podName, containerName, filePath string) (*models.ListFilesResponse, error) {
	return ListFiles(appCtx, podName, containerName, filePath)
}

var readBuilderWorkspaceFileInContainer = func(appCtx *models.AppContext, podName, containerName, filePath string) (*models.ReadFileResponse, error) {
	return ReadFile(appCtx, podName, containerName, filePath)
}

var downloadBuilderWorkspaceArchive = func(appCtx *models.AppContext, podName, containerName, workspaceRoot string, writer io.Writer) error {
	return execCommandStreamStdout(
		appCtx,
		podName,
		containerName,
		[]string{"tar", "czf", "-", "-C", path.Clean(workspaceRoot), "."},
		writer,
	)
}

func ProvisionBuilderWorkspace(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
	tx := db.DB.WithContext(ctx)

	session, err := loadBuilderSession(tx, "", sessionID)
	if err != nil {
		return nil, err
	}

	workspace, err := getCurrentBuilderWorkspace(tx, session.ID)
	if err == nil {
		if updateErr := markBuilderSessionReady(tx, session.ID); updateErr != nil {
			return nil, updateErr
		}
		return workspace, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	buildEnv, err := GetEnv(session.BuildEnvID)
	if err != nil {
		return nil, err
	}
	project, err := GetProject(session.ProjectID)
	if err != nil {
		return nil, err
	}

	resources, err := core.BuildBuilderWorkspaceResources(core.BuilderWorkspaceSpec{
		SessionID:      session.ID,
		ProjectID:      session.ProjectID,
		ProjectSlug:    project.Slug,
		BuildEnvID:     buildEnv.ID,
		BuildEnvSlug:   buildEnv.Slug,
		Namespace:      buildEnv.ClusterNamespace,
		StorageRequest: builderWorkspaceStorageRequest,
	})
	if err != nil {
		return nil, err
	}

	client, err := getBuilderWorkspaceClusterClient(buildEnv.ClusterID)
	if err != nil {
		return nil, err
	}

	if _, err := client.CoreV1().PersistentVolumeClaims(buildEnv.ClusterNamespace).Create(ctx, resources.PersistentVolumeClaim, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}
	if _, err := client.CoreV1().Pods(buildEnv.ClusterNamespace).Create(ctx, resources.Pod, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	now := time.Now().UTC()
	workspace = &entities.BuilderWorkspace{
		ID:            uuid.New(),
		CreatedAt:     now,
		UpdatedAt:     now,
		SessionID:     session.ID,
		BuildEnvID:    buildEnv.ID,
		ClusterID:     buildEnv.ClusterID,
		Namespace:     buildEnv.ClusterNamespace,
		PodName:       resources.Pod.Name,
		ContainerName: resources.Pod.Spec.Containers[0].Name,
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: app.Config.BuilderWorkspaceRoot,
	}

	if err := tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(workspace).Error; err != nil {
			return err
		}
		return markBuilderSessionReady(tx, session.ID)
	}); err != nil {
		return nil, err
	}

	return workspace, nil
}

func ListBuilderWorkspaceFiles(ctx context.Context, projectID, sessionID, requestedPath string) (*models.ListFilesResponse, error) {
	workspace, err := getBuilderWorkspaceForProjectSession(ctx, projectID, sessionID)
	if err != nil {
		return nil, err
	}

	resolvedPath, err := resolveBuilderWorkspacePath(workspace.WorkspaceRoot, requestedPath)
	if err != nil {
		return nil, err
	}

	appCtx, err := buildBuilderWorkspaceAppContext(workspace)
	if err != nil {
		return nil, err
	}

	return listBuilderWorkspaceFilesInContainer(appCtx, workspace.PodName, workspace.ContainerName, resolvedPath)
}

func ReadBuilderWorkspaceFile(ctx context.Context, projectID, sessionID, requestedPath string) (*models.ReadFileResponse, error) {
	workspace, err := getBuilderWorkspaceForProjectSession(ctx, projectID, sessionID)
	if err != nil {
		return nil, err
	}

	resolvedPath, err := resolveBuilderWorkspacePath(workspace.WorkspaceRoot, requestedPath)
	if err != nil {
		return nil, err
	}

	appCtx, err := buildBuilderWorkspaceAppContext(workspace)
	if err != nil {
		return nil, err
	}

	return readBuilderWorkspaceFileInContainer(appCtx, workspace.PodName, workspace.ContainerName, resolvedPath)
}

func DownloadBuilderWorkspace(ctx context.Context, projectID, sessionID string, writer io.Writer) error {
	workspace, err := getBuilderWorkspaceForProjectSession(ctx, projectID, sessionID)
	if err != nil {
		return err
	}

	appCtx, err := buildBuilderWorkspaceAppContext(workspace)
	if err != nil {
		return err
	}

	return downloadBuilderWorkspaceArchive(appCtx, workspace.PodName, workspace.ContainerName, workspace.WorkspaceRoot, writer)
}

func writeBuilderAgentFiles(ctx context.Context, workspace *entities.BuilderWorkspace, run *entities.BuilderRun, files []BuilderAgentFileWrite) error {
	if workspace == nil {
		return errors.New("builder workspace is required")
	}
	if run == nil {
		return errors.New("builder run is required")
	}

	appCtx, err := buildBuilderWorkspaceAppContext(workspace)
	if err != nil {
		return err
	}

	for _, file := range files {
		validatedFile, err := validateBuilderAgentFileWrite(file)
		if err != nil {
			return err
		}

		resolvedPath, err := resolveBuilderWorkspacePath(workspace.WorkspaceRoot, validatedFile.Path)
		if err != nil {
			return err
		}

		if err := writeBuilderWorkspaceFile(appCtx, workspace.PodName, workspace.ContainerName, resolvedPath, validatedFile.Content); err != nil {
			return err
		}
	}

	return refreshBuilderWorkspaceArtifacts(ctx, appCtx, workspace, run)
}

func refreshBuilderWorkspaceArtifacts(ctx context.Context, appCtx *models.AppContext, workspace *entities.BuilderWorkspace, run *entities.BuilderRun) error {
	listing, err := listBuilderWorkspaceFilesInContainer(appCtx, workspace.PodName, workspace.ContainerName, workspace.WorkspaceRoot)
	if err != nil {
		return err
	}

	artifacts := make([]entities.BuilderArtifact, 0, len(listing.Files))
	for _, file := range listing.Files {
		if file.Type != "file" {
			continue
		}

		artifactPath, err := validateBuilderAgentFilePath(file.Name)
		if err != nil {
			return err
		}

		metadataJSON, err := json.Marshal(map[string]int64{"size_bytes": file.Size})
		if err != nil {
			return err
		}

		artifacts = append(artifacts, entities.BuilderArtifact{
			ID:           uuid.New(),
			SessionID:    workspace.SessionID,
			WorkspaceID:  workspace.ID,
			RunID:        run.ID,
			Kind:         "file",
			Path:         artifactPath,
			MetadataJSON: string(metadataJSON),
		})
	}

	tx := db.DB.WithContext(ctx)
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ? AND workspace_id = ?", workspace.SessionID, workspace.ID).Delete(&entities.BuilderArtifact{}).Error; err != nil {
			return err
		}
		if len(artifacts) == 0 {
			return nil
		}
		return tx.Create(&artifacts).Error
	})
}

func buildBuilderWorkspaceAppContext(workspace *entities.BuilderWorkspace) (*models.AppContext, error) {
	buildEnv, err := GetEnv(workspace.BuildEnvID)
	if err != nil {
		return nil, err
	}
	cluster, err := GetCluster(workspace.ClusterID)
	if err != nil {
		return nil, err
	}
	project, err := GetProject(buildEnv.ProjectID)
	if err != nil {
		return nil, err
	}

	envCopy := *buildEnv
	envCopy.ClusterID = workspace.ClusterID
	envCopy.ClusterNamespace = workspace.Namespace

	return &models.AppContext{
		EnvContext: models.EnvContext{
			Env:     envCopy,
			Project: *project,
			Cluster: *cluster,
		},
	}, nil
}

func getBuilderWorkspaceForProjectSession(ctx context.Context, projectID, sessionID string) (*entities.BuilderWorkspace, error) {
	tx := db.DB.WithContext(ctx)
	if _, err := loadBuilderSession(tx, projectID, sessionID); err != nil {
		return nil, err
	}
	return getCurrentBuilderWorkspace(tx, sessionID)
}

func getCurrentBuilderWorkspace(tx *gorm.DB, sessionID string) (*entities.BuilderWorkspace, error) {
	var workspace entities.BuilderWorkspace
	if err := tx.Where("session_id = ? AND terminated_at IS NULL", sessionID).
		Order("created_at DESC, id DESC").
		Take(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func markBuilderSessionReady(tx *gorm.DB, sessionID string) error {
	now := time.Now().UTC()
	return tx.Model(&entities.BuilderSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"status":     entities.BuilderSessionStatusReady,
			"updated_at": now,
		}).Error
}

func resolveBuilderWorkspacePath(workspaceRoot, requestedPath string) (string, error) {
	cleanWorkspaceRoot := path.Clean(strings.TrimSpace(strings.ReplaceAll(workspaceRoot, "\\", "/")))
	if cleanWorkspaceRoot == "" || cleanWorkspaceRoot == "." {
		return "", errors.New("builder workspace root is required")
	}

	normalizedRequestedPath := strings.TrimSpace(strings.ReplaceAll(requestedPath, "\\", "/"))
	if normalizedRequestedPath == "" || normalizedRequestedPath == "." || normalizedRequestedPath == "/" {
		return cleanWorkspaceRoot, nil
	}
	if strings.HasPrefix(normalizedRequestedPath, "~") {
		return "", fmt.Errorf("%w: %s", ErrBuilderAgentUnsafeFilePath, requestedPath)
	}

	var candidatePath string
	if path.IsAbs(normalizedRequestedPath) {
		candidatePath = path.Clean(normalizedRequestedPath)
	} else {
		candidatePath = path.Clean(path.Join(cleanWorkspaceRoot, normalizedRequestedPath))
	}

	if candidatePath != cleanWorkspaceRoot && !strings.HasPrefix(candidatePath, cleanWorkspaceRoot+"/") {
		return "", fmt.Errorf("%w: %s", ErrBuilderAgentUnsafeFilePath, requestedPath)
	}

	return candidatePath, nil
}

var (
	builderRunBroadcasters sync.Map
)

func subscribeBuilderRunLogs(runID string) (<-chan string, func()) {
	ch := make(chan string, 256)
	builderRunBroadcasters.Store(runID, ch)
	return ch, func() {
		builderRunBroadcasters.Delete(runID)
		close(ch)
	}
}

func publishBuilderRunLog(runID string, line string) {
	if ch, ok := builderRunBroadcasters.Load(runID); ok {
		select {
		case ch.(chan string) <- line:
		default:
		}
	}
}

func ExecuteNextBuilderRun(ctx context.Context, sessionID string) error {
	tx := db.DB.WithContext(ctx)

	session, err := loadBuilderSession(tx, "", sessionID)
	if err != nil {
		return err
	}

	workspace, err := ProvisionBuilderWorkspace(ctx, sessionID)
	if err != nil {
		markBuilderRunFailed(ctx, tx, sessionID, "", "workspace provisioning failed: "+err.Error())
		markSessionFailed(ctx, tx, sessionID)
		return err
	}

	run, err := claimAndStartNextBuilderRun(ctx, tx, session, workspace)
	if err != nil {
		return err
	}
	if run == nil {
		return nil
	}

	go executeBuilderRun(context.Background(), session, workspace, run)
	return nil
}

func executeBuilderRun(ctx context.Context, session *entities.BuilderSession, workspace *entities.BuilderWorkspace, run *entities.BuilderRun) {
	runID := run.ID
	sessionID := session.ID

	defer func() {
		finalizeBuilderRun(context.Background(), sessionID, runID)
	}()

	publishBuilderRunLog(runID, "[system] run started\n")

	messages, err := loadBuilderConversationMessages(ctx, sessionID)
	if err != nil {
		publishBuilderRunLog(runID, "[system] failed to load conversation: "+err.Error()+"\n")
		return
	}

	publishBuilderRunLog(runID, "[agent] generating files...\n")

	result, err := GenerateBuilderFiles(ctx, messages)
	if err != nil {
		publishBuilderRunLog(runID, "[agent] error: "+err.Error()+"\n")
		appendBuilderSessionSystemMessage(ctx, sessionID, runID, "run failed: "+err.Error())
		markBuilderRunFailed(ctx, nil, sessionID, runID, err.Error())
		markSessionReadyOrFailed(ctx, sessionID, false)
		return
	}

	if len(result.Files) == 0 {
		publishBuilderRunLog(runID, "[agent] no files to write\n")
		appendBuilderSessionAssistantMessage(ctx, sessionID, runID, result.AssistantMessage)
		appendBuilderSessionSystemMessage(ctx, sessionID, runID, "run completed: no files generated")
		persistBuilderRunSuccess(ctx, sessionID, runID, "", result.AssistantMessage)
		markSessionReadyOrFailed(ctx, sessionID, true)
		return
	}

	for _, file := range result.Files {
		publishBuilderRunLog(runID, "[agent] writing "+file.Path+"\n")
	}

	if err := writeBuilderAgentFiles(ctx, workspace, run, result.Files); err != nil {
		publishBuilderRunLog(runID, "[system] file write failed: "+err.Error()+"\n")
		appendBuilderSessionSystemMessage(ctx, sessionID, runID, "run failed: "+err.Error())
		markBuilderRunFailed(ctx, nil, sessionID, runID, err.Error())
		markSessionReadyOrFailed(ctx, sessionID, false)
		return
	}

	publishBuilderRunLog(runID, "[system] "+fmt.Sprintf("%d files written\n", len(result.Files)))

	appendBuilderSessionAssistantMessage(ctx, sessionID, runID, result.AssistantMessage)
	appendBuilderSessionSystemMessage(ctx, sessionID, runID, fmt.Sprintf("run completed: %d files generated", len(result.Files)))

	executionLog := buildExecutionLogSummary(runID, result)
	persistBuilderRunSuccess(ctx, sessionID, runID, executionLog, result.AssistantMessage)
	markSessionReadyOrFailed(ctx, sessionID, true)
}

func claimAndStartNextBuilderRun(ctx context.Context, tx *gorm.DB, session *entities.BuilderSession, workspace *entities.BuilderWorkspace) (*entities.BuilderRun, error) {
	var run entities.BuilderRun
	if err := tx.Where("session_id = ? AND status = ?", session.ID, entities.BuilderRunStatusQueued).
		Order("created_at ASC, id ASC").
		Take(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	now := time.Now().UTC()
	if err := tx.Model(&run).Updates(map[string]any{
		"status":       entities.BuilderRunStatusExecuting,
		"workspace_id": workspaceIDPtr(workspace.ID),
		"started_at":   now,
	}).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&entities.BuilderSession{}).Where("id = ?", session.ID).Updates(map[string]any{
		"status":           entities.BuilderSessionStatusRunning,
		"updated_at":       now,
		"last_activity_at": now,
	}).Error; err != nil {
		return nil, err
	}

	run.Status = entities.BuilderRunStatusExecuting
	run.StartedAt = &now
	run.WorkspaceID = workspaceIDPtr(workspace.ID)
	return &run, nil
}

func finalizeBuilderRun(ctx context.Context, sessionID, runID string) {
	if err := triggerNextBuilderRun(context.Background(), sessionID); err != nil {
		return
	}
}

func triggerNextBuilderRun(ctx context.Context, sessionID string) error {
	tx := db.DB.WithContext(ctx)
	var nextRun entities.BuilderRun
	if err := tx.Where("session_id = ? AND status = ?", sessionID, entities.BuilderRunStatusQueued).
		Order("created_at ASC, id ASC").
		Take(&nextRun).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	go ExecuteNextBuilderRun(context.Background(), sessionID)
	return nil
}

func loadBuilderConversationMessages(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
	var messages []entities.BuilderMessage
	if err := db.DB.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at ASC, id ASC").Find(&messages).Error; err != nil {
		return nil, err
	}

	agentMsgs := make([]BuilderAgentMessage, 0, len(messages))
	for _, m := range messages {
		agentMsgs = append(agentMsgs, BuilderAgentMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}
	return agentMsgs, nil
}

func appendBuilderSessionAssistantMessage(ctx context.Context, sessionID string, runID string, content string) {
	db.DB.WithContext(ctx).Create(&entities.BuilderMessage{
		ID:        uuid.New(),
		SessionID: sessionID,
		RunID:     &runID,
		Role:      entities.BuilderMessageRoleAssistant,
		Content:   content,
		CreatedBy: "builder-agent",
	})
}

func appendBuilderSessionSystemMessage(ctx context.Context, sessionID string, runID string, content string) {
	db.DB.WithContext(ctx).Create(&entities.BuilderMessage{
		ID:        uuid.New(),
		SessionID: sessionID,
		RunID:     &runID,
		Role:      entities.BuilderMessageRoleSystem,
		Content:   content,
		CreatedBy: "builder-agent",
	})
}

func markBuilderRunFailed(ctx context.Context, tx *gorm.DB, sessionID, runID string, errMsg string) {
	if tx == nil {
		tx = db.DB.WithContext(ctx)
	}
	now := time.Now().UTC()
	tx.Model(&entities.BuilderRun{}).Where("id = ? AND session_id = ?", runID, sessionID).Updates(map[string]any{
		"status":        entities.BuilderRunStatusFailed,
		"error_message": errMsg,
		"completed_at":  now,
	})
}

func markSessionFailed(ctx context.Context, tx *gorm.DB, sessionID string) {
	now := time.Now().UTC()
	tx.Model(&entities.BuilderSession{}).Where("id = ?", sessionID).Updates(map[string]any{
		"status":     entities.BuilderSessionStatusFailed,
		"updated_at": now,
	})
}

func markSessionReadyOrFailed(ctx context.Context, sessionID string, succeeded bool) {
	tx := db.DB.WithContext(ctx)
	now := time.Now().UTC()
	status := entities.BuilderSessionStatusReady
	if !succeeded {
		status = entities.BuilderSessionStatusFailed
	}
	tx.Model(&entities.BuilderSession{}).Where("id = ?", sessionID).Updates(map[string]any{
		"status":           status,
		"updated_at":       now,
		"last_activity_at": now,
	})
}

func persistBuilderRunSuccess(ctx context.Context, sessionID, runID, executionLog, assistantMessage string) {
	tx := db.DB.WithContext(ctx)
	now := time.Now().UTC()
	tx.Model(&entities.BuilderRun{}).Where("id = ? AND session_id = ?", runID, sessionID).Updates(map[string]any{
		"status":        entities.BuilderRunStatusSucceeded,
		"execution_log": executionLog,
		"completed_at":  now,
	})
}

func buildExecutionLogSummary(runID string, result *BuilderAgentResult) string {
	lines := make([]string, 0, len(result.Files)+2)
	lines = append(lines, "[agent] response: "+result.AssistantMessage)
	for _, f := range result.Files {
		lines = append(lines, "[agent] wrote: "+f.Path)
	}
	return strings.Join(lines, "\n")
}

func workspaceIDPtr(id string) *string {
	return &id
}

func waitForBuilderWorkspacePodReady(ctx context.Context, client kubernetes.Interface, namespace, podName string) error {
	return wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}
