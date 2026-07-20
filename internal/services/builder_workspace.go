package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
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

const (
	builderWorkspaceStorageRequest = "5Gi"
	builderBuildOutputRootDist     = "dist"
	builderBuildOutputRootBuild    = "build"
	builderBuildOutputRootNext     = ".next"
)

type builderArtifactMetadata struct {
	SizeBytes  int64  `json:"size_bytes"`
	OutputRoot string `json:"output_root,omitempty"`
}

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
	executionImage, err := loadBuilderWorkspaceExecutionImage(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return provisionBuilderWorkspace(ctx, sessionID, executionImage)
}

func provisionBuilderWorkspace(ctx context.Context, sessionID, executionImage string) (*entities.BuilderWorkspace, error) {
	tx := db.DB.WithContext(ctx)

	session, err := loadBuilderSession(tx, "", sessionID)
	if err != nil {
		return nil, err
	}

	workspace, err := getCurrentBuilderWorkspace(tx, session.ID)
	workspaceExists := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
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

	handles, err := listCurrentBuilderWorkspaceExecutorHandles(tx, session.ID)
	if err != nil {
		return nil, err
	}

	orderedHandles := orderBuilderWorkspaceExecutorHandlesForWorkspace(workspace, handles)
	var canonicalHandle *entities.BuilderExecutorHandle
	var canonicalSnapshot *builderExecutorSnapshot
	var ensureHandle *entities.BuilderExecutorHandle

	for i := range orderedHandles {
		handle := &orderedHandles[i]
		if !builderWorkspaceExecutorHandleHasSnapshotRuntimeRefs(handle) {
			if ensureHandle == nil {
				ensureHandle = handle
			}
			continue
		}

		executor, err := getBuilderWorkspaceExecutor(handle.Kind)
		if err != nil {
			return nil, err
		}

		snapshot, err := executor.GetWorkspaceSnapshot(ctx, handle)
		if err != nil {
			return nil, err
		}
		if snapshot == nil {
			continue
		}
		if snapshot.Usable() {
			canonicalHandle = handle
			canonicalSnapshot = snapshot
			break
		}
		if !snapshot.Terminal() && ensureHandle == nil {
			ensureHandle = handle
		}
	}

	if canonicalHandle == nil {
		if ensureHandle == nil {
			ensureHandle, err = createBuilderWorkspaceExecutorHandle(tx, session.ID, buildEnv.ClusterID, buildEnv.ClusterNamespace)
			if err != nil {
				return nil, err
			}
			orderedHandles = append(orderedHandles, *ensureHandle)
		}

		executor, err := getBuilderWorkspaceExecutor(ensureHandle.Kind)
		if err != nil {
			return nil, err
		}

		canonicalSnapshot, err = executor.EnsureWorkspace(ctx, builderWorkspaceExecutorRequest{
			Handle:         ensureHandle,
			SessionID:      session.ID,
			ProjectID:      session.ProjectID,
			ProjectSlug:    project.Slug,
			BuildEnvID:     buildEnv.ID,
			BuildEnvSlug:   buildEnv.Slug,
			ClusterID:      buildEnv.ClusterID,
			Namespace:      buildEnv.ClusterNamespace,
			ExecutionImage: executionImage,
			StorageRequest: builderWorkspaceStorageRequest,
		})
		if err != nil {
			return nil, err
		}
		canonicalHandle = ensureHandle
	}

	now := time.Now().UTC()
	if !workspaceExists || workspace == nil {
		workspace = &entities.BuilderWorkspace{
			ID:        uuid.New(),
			CreatedAt: now,
			SessionID: session.ID,
		}
	}
	workspace.UpdatedAt = now

	if err := reconcileBuilderWorkspaceFromExecutorSnapshot(workspace, buildEnv.ID, app.Config.BuilderWorkspaceRoot, canonicalSnapshot); err != nil {
		return nil, err
	}

	if err := tx.Transaction(func(tx *gorm.DB) error {
		if err := persistBuilderWorkspaceExecutorHandleSnapshot(tx, canonicalHandle, canonicalSnapshot); err != nil {
			return err
		}
		for i := range orderedHandles {
			handle := &orderedHandles[i]
			if canonicalHandle != nil && handle.ID == canonicalHandle.ID {
				continue
			}
			if handle.TerminatedAt != nil || handle.Status == entities.BuilderExecutorHandleStatusTerminated {
				continue
			}

			if builderWorkspaceExecutorHandleHasSnapshotRuntimeRefs(handle) && !builderWorkspaceExecutorHandleMatchesSnapshot(handle, canonicalSnapshot) {
				executor, err := getBuilderWorkspaceExecutor(handle.Kind)
				if err != nil {
					return err
				}
				if err := executor.CancelWorkspace(ctx, handle); err != nil {
					return err
				}
			}
			if err := markBuilderWorkspaceExecutorHandleTerminated(tx, handle.ID); err != nil {
				return err
			}
		}
		if workspaceExists {
			if err := tx.Save(workspace).Error; err != nil {
				return err
			}
		} else if err := tx.Create(workspace).Error; err != nil {
			return err
		}
		return markBuilderSessionReady(tx, session.ID)
	}); err != nil {
		return nil, err
	}

	return workspace, nil
}

func loadBuilderWorkspaceExecutionImage(ctx context.Context, sessionID string) (string, error) {
	if strings.TrimSpace(sessionID) == "" {
		return "", errors.New("builder session id is required")
	}

	var run entities.BuilderRun
	err := db.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC, id DESC").
		First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(stringPointerValue(run.ExecutionImageRef)), nil
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
		if run.ClaimToken != nil && strings.TrimSpace(*run.ClaimToken) != "" {
			if err := ensureClaimedBuilderRunOwnership(ctx, run.ID, *run.ClaimToken); err != nil {
				return err
			}
		}

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

	if run.ClaimToken != nil && strings.TrimSpace(*run.ClaimToken) != "" {
		if err := ensureClaimedBuilderRunOwnership(ctx, run.ID, *run.ClaimToken); err != nil {
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

	artifacts, err := collectBuilderWorkspaceFileArtifacts(workspace, run, listing)
	if err != nil {
		return err
	}

	if run.ClaimToken != nil && strings.TrimSpace(*run.ClaimToken) != "" {
		return withOwnedExecutingBuilderRunTx(ctx, run.ID, *run.ClaimToken, func(tx *gorm.DB, _ *entities.BuilderRun) error {
			return replaceBuilderWorkspaceArtifactsTx(tx, workspace, artifacts)
		})
	}

	tx := db.DB.WithContext(ctx)
	return tx.Transaction(func(tx *gorm.DB) error {
		return replaceBuilderWorkspaceArtifactsTx(tx, workspace, artifacts)
	})
}

func CollectBuilderBuildArtifacts(workspace *entities.BuilderWorkspace, run *entities.BuilderRun, listing *models.ListFilesResponse) ([]entities.BuilderArtifact, error) {
	appCtx, err := buildBuilderWorkspaceAppContext(workspace)
	if err != nil {
		return nil, err
	}

	return collectBuilderBuildArtifactsWithListFn(context.Background(), workspace, run, listing, func(_ context.Context, workspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
		return listBuilderWorkspaceFilesInContainer(appCtx, workspace.PodName, workspace.ContainerName, requestedPath)
	})
}

func collectBuilderBuildArtifactsWithListFn(ctx context.Context, workspace *entities.BuilderWorkspace, run *entities.BuilderRun, listing *models.ListFilesResponse, listFiles func(context.Context, *entities.BuilderWorkspace, string) (*models.ListFilesResponse, error)) ([]entities.BuilderArtifact, error) {
	outputRoot, err := detectBuilderBuildOutputRoot(workspace, listing)
	if err != nil {
		return nil, err
	}
	if listFiles == nil {
		return nil, errors.New("builder artifact list function is required")
	}

	return collectBuilderBuildArtifactsRecursive(ctx, workspace, run, listing, outputRoot, listFiles)
}

func collectBuilderWorkspaceFileArtifacts(workspace *entities.BuilderWorkspace, run *entities.BuilderRun, listing *models.ListFilesResponse) ([]entities.BuilderArtifact, error) {
	return collectBuilderArtifactsFromListing(workspace, run, listing, entities.BuilderArtifactKindWorkspaceFile, "")
}

func collectBuilderArtifactsFromListing(workspace *entities.BuilderWorkspace, run *entities.BuilderRun, listing *models.ListFilesResponse, kind entities.BuilderArtifactKind, outputRoot string) ([]entities.BuilderArtifact, error) {
	if workspace == nil {
		return nil, errors.New("builder workspace is required")
	}
	if run == nil {
		return nil, errors.New("builder run is required")
	}
	if listing == nil {
		return nil, errors.New("builder artifact listing is required")
	}

	artifactBasePath, err := resolveBuilderArtifactListingBasePath(workspace.WorkspaceRoot, listing.Path)
	if err != nil {
		return nil, err
	}

	artifacts := make([]entities.BuilderArtifact, 0, len(listing.Files))
	for _, file := range listing.Files {
		if file.Type != "file" {
			continue
		}

		artifact, err := newBuilderArtifact(workspace, run, kind, path.Join(artifactBasePath, file.Name), file.Size, outputRoot)
		if err != nil {
			return nil, err
		}

		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func collectBuilderBuildArtifactsRecursive(ctx context.Context, workspace *entities.BuilderWorkspace, run *entities.BuilderRun, listing *models.ListFilesResponse, outputRoot string, listFiles func(context.Context, *entities.BuilderWorkspace, string) (*models.ListFilesResponse, error)) ([]entities.BuilderArtifact, error) {
	if workspace == nil {
		return nil, errors.New("builder workspace is required")
	}
	if run == nil {
		return nil, errors.New("builder run is required")
	}
	if listing == nil {
		return nil, errors.New("builder artifact listing is required")
	}
	if listFiles == nil {
		return nil, errors.New("builder artifact list function is required")
	}

	artifactBasePath, err := resolveBuilderArtifactListingBasePath(workspace.WorkspaceRoot, listing.Path)
	if err != nil {
		return nil, err
	}

	artifacts := make([]entities.BuilderArtifact, 0, len(listing.Files))
	for _, file := range listing.Files {
		switch file.Type {
		case "file":
			artifact, err := newBuilderArtifact(workspace, run, entities.BuilderArtifactKindBuildOutput, path.Join(artifactBasePath, file.Name), file.Size, outputRoot)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, artifact)
		case "dir":
			nestedListingPath := path.Join(listing.Path, file.Name)
			nestedListing, err := listFiles(ctx, workspace, nestedListingPath)
			if err != nil {
				return nil, err
			}
			nestedArtifacts, err := collectBuilderBuildArtifactsRecursive(ctx, workspace, run, nestedListing, outputRoot, listFiles)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, nestedArtifacts...)
		}
	}

	return artifacts, nil
}

func detectBuilderBuildOutputRoot(workspace *entities.BuilderWorkspace, listing *models.ListFilesResponse) (string, error) {
	if workspace == nil {
		return "", errors.New("builder workspace is required")
	}
	if listing == nil {
		return "", errors.New("builder artifact listing is required")
	}

	artifactBasePath, err := resolveBuilderArtifactListingBasePath(workspace.WorkspaceRoot, listing.Path)
	if err != nil {
		return "", err
	}
	if artifactBasePath == "" {
		return "", errors.New("builder build output listing must be rooted under a supported output directory")
	}

	segments := strings.Split(artifactBasePath, "/")
	if len(segments) == 0 {
		return "", errors.New("builder build output listing must be rooted under a supported output directory")
	}

	switch segments[0] {
	case builderBuildOutputRootDist, builderBuildOutputRootBuild, builderBuildOutputRootNext:
		return segments[0], nil
	default:
		return "", app.NewErrorf("unsupported builder build output root: %s", artifactBasePath)
	}
}

func resolveBuilderArtifactListingBasePath(workspaceRoot, listingPath string) (string, error) {
	cleanWorkspaceRoot := path.Clean(strings.TrimSpace(strings.ReplaceAll(workspaceRoot, "\\", "/")))
	resolvedListingPath, err := resolveBuilderWorkspacePath(workspaceRoot, listingPath)
	if err != nil {
		return "", err
	}
	if resolvedListingPath == cleanWorkspaceRoot {
		return "", nil
	}
	return strings.TrimPrefix(resolvedListingPath, cleanWorkspaceRoot+"/"), nil
}

func marshalBuilderArtifactMetadata(sizeBytes int64, outputRoot string) (string, error) {
	metadataJSON, err := json.Marshal(builderArtifactMetadata{
		SizeBytes:  sizeBytes,
		OutputRoot: outputRoot,
	})
	if err != nil {
		return "", err
	}
	return string(metadataJSON), nil
}

func newBuilderArtifact(workspace *entities.BuilderWorkspace, run *entities.BuilderRun, kind entities.BuilderArtifactKind, artifactPath string, sizeBytes int64, outputRoot string) (entities.BuilderArtifact, error) {
	validatedArtifactPath, err := validateBuilderAgentFilePath(artifactPath)
	if err != nil {
		return entities.BuilderArtifact{}, err
	}

	metadataJSON, err := marshalBuilderArtifactMetadata(sizeBytes, outputRoot)
	if err != nil {
		return entities.BuilderArtifact{}, err
	}

	return entities.BuilderArtifact{
		ID:           uuid.New(),
		SessionID:    workspace.SessionID,
		WorkspaceID:  workspace.ID,
		RunID:        run.ID,
		Kind:         kind,
		Path:         validatedArtifactPath,
		MetadataJSON: metadataJSON,
	}, nil
}

func replaceBuilderWorkspaceArtifactsTx(tx *gorm.DB, workspace *entities.BuilderWorkspace, artifacts []entities.BuilderArtifact) error {
	if err := tx.Where("session_id = ? AND workspace_id = ?", workspace.SessionID, workspace.ID).Delete(&entities.BuilderArtifact{}).Error; err != nil {
		return err
	}
	if len(artifacts) == 0 {
		return nil
	}
	return tx.Create(&artifacts).Error
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
		PodAccessPolicy: &models.PodAccessPolicy{RequiredLabels: map[string]string{
			kube.LabelBuilderWorkspace: "true",
			kube.LabelBuilderSessionID: workspace.SessionID,
		}},
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

func listCurrentBuilderWorkspaceExecutorHandles(tx *gorm.DB, sessionID string) ([]entities.BuilderExecutorHandle, error) {
	handles := make([]entities.BuilderExecutorHandle, 0)
	if err := tx.Where("session_id = ? AND kind = ? AND status IN ? AND terminated_at IS NULL", sessionID, entities.BuilderExecutorHandleKindWorkspacePod, []entities.BuilderExecutorHandleStatus{entities.BuilderExecutorHandleStatusProvisioning, entities.BuilderExecutorHandleStatusActive}).
		Order("created_at DESC, id DESC").
		Find(&handles).Error; err != nil {
		return nil, err
	}
	return handles, nil
}

func orderBuilderWorkspaceExecutorHandlesForWorkspace(workspace *entities.BuilderWorkspace, handles []entities.BuilderExecutorHandle) []entities.BuilderExecutorHandle {
	if workspace == nil || workspace.ExecutorHandleID == nil {
		return handles
	}

	ordered := make([]entities.BuilderExecutorHandle, 0, len(handles))
	for i := range handles {
		if handles[i].ID == *workspace.ExecutorHandleID {
			ordered = append(ordered, handles[i])
			break
		}
	}
	for i := range handles {
		if handles[i].ID == *workspace.ExecutorHandleID {
			continue
		}
		ordered = append(ordered, handles[i])
	}
	return ordered
}

func createBuilderWorkspaceExecutorHandle(tx *gorm.DB, sessionID, clusterID, namespace string) (*entities.BuilderExecutorHandle, error) {
	now := time.Now().UTC()
	handle := &entities.BuilderExecutorHandle{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		SessionID: sessionID,
		Kind:      entities.BuilderExecutorHandleKindWorkspacePod,
		Status:    entities.BuilderExecutorHandleStatusProvisioning,
		ClusterID: builderStringPtr(clusterID),
		Namespace: builderStringPtr(namespace),
	}
	if err := tx.Create(handle).Error; err != nil {
		return nil, err
	}
	return handle, nil
}

func persistBuilderWorkspaceExecutorHandleSnapshot(tx *gorm.DB, handle *entities.BuilderExecutorHandle, snapshot *builderExecutorSnapshot) error {
	if handle == nil {
		return errors.New("builder executor handle is required")
	}
	if snapshot == nil {
		return errors.New("builder executor snapshot is required")
	}

	now := time.Now().UTC()
	handle.UpdatedAt = now
	handle.Kind = snapshot.Kind
	handle.Status = snapshot.Status
	handle.ClusterID = builderStringPtr(snapshot.ClusterID)
	handle.Namespace = builderStringPtr(snapshot.Namespace)
	handle.WorkloadName = builderStringPtr(snapshot.WorkloadName)
	handle.ContainerName = builderStringPtr(snapshot.ContainerName)

	return tx.Model(&entities.BuilderExecutorHandle{}).
		Where("id = ?", handle.ID).
		Updates(map[string]any{
			"updated_at":     now,
			"kind":           handle.Kind,
			"status":         handle.Status,
			"cluster_id":     handle.ClusterID,
			"namespace":      handle.Namespace,
			"workload_name":  handle.WorkloadName,
			"container_name": handle.ContainerName,
		}).Error
}

func markBuilderWorkspaceExecutorHandleTerminated(tx *gorm.DB, handleID string) error {
	now := time.Now().UTC()
	return tx.Model(&entities.BuilderExecutorHandle{}).
		Where("id = ?", handleID).
		Updates(map[string]any{
			"status":        entities.BuilderExecutorHandleStatusTerminated,
			"updated_at":    now,
			"terminated_at": now,
		}).Error
}

func builderWorkspaceExecutorHandleMatchesSnapshot(handle *entities.BuilderExecutorHandle, snapshot *builderExecutorSnapshot) bool {
	if handle == nil || snapshot == nil {
		return false
	}
	if handle.ClusterID == nil || handle.Namespace == nil || handle.WorkloadName == nil {
		return false
	}
	if *handle.ClusterID != snapshot.ClusterID || *handle.Namespace != snapshot.Namespace || *handle.WorkloadName != snapshot.WorkloadName {
		return false
	}
	return true
}

func builderWorkspaceExecutorHandleHasSnapshotRuntimeRefs(handle *entities.BuilderExecutorHandle) bool {
	if handle == nil {
		return false
	}
	if handle.ClusterID == nil || *handle.ClusterID == "" {
		return false
	}
	if handle.Namespace == nil || *handle.Namespace == "" {
		return false
	}
	if handle.WorkloadName == nil || *handle.WorkloadName == "" {
		return false
	}
	return true
}

func markBuilderSessionReady(tx *gorm.DB, sessionID string) error {
	now := time.Now().UTC()
	return tx.Model(&entities.BuilderSession{}).
		Where("id = ? AND status IN ?", sessionID, []entities.BuilderSessionStatus{
			entities.BuilderSessionStatusProvisioning,
			entities.BuilderSessionStatusReady,
		}).
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
		return "", app.WrapErrorf(ErrBuilderAgentUnsafeFilePath, "%w: %s", ErrBuilderAgentUnsafeFilePath, requestedPath)
	}

	var candidatePath string
	if path.IsAbs(normalizedRequestedPath) {
		candidatePath = path.Clean(normalizedRequestedPath)
	} else {
		candidatePath = path.Clean(path.Join(cleanWorkspaceRoot, normalizedRequestedPath))
	}

	if candidatePath != cleanWorkspaceRoot && !strings.HasPrefix(candidatePath, cleanWorkspaceRoot+"/") {
		return "", app.WrapErrorf(ErrBuilderAgentUnsafeFilePath, "%w: %s", ErrBuilderAgentUnsafeFilePath, requestedPath)
	}

	return candidatePath, nil
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

func appendBuilderSessionMessageTx(tx *gorm.DB, sessionID string, runID string, role entities.BuilderMessageRole, content string) error {
	return tx.Create(&entities.BuilderMessage{
		ID:        uuid.New(),
		SessionID: sessionID,
		RunID:     &runID,
		Role:      role,
		Content:   content,
		CreatedBy: "builder-agent",
	}).Error
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
