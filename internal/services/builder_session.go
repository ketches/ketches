package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

var (
	ErrBuilderSessionNotFound      = errors.New("builder session not found")
	ErrBuilderSessionNotAppendable = errors.New("builder session not appendable")
	ErrBuilderRunNotFound          = errors.New("builder run not found")
	builderWorkerQueuedRunNotifier = func() {
		GlobalBuilderWorker.Nudge()
	}
)

type BuilderSessionNotFoundError struct {
	ProjectID string
	SessionID string
}

func (e *BuilderSessionNotFoundError) Error() string {
	if e.ProjectID == "" {
		return fmt.Sprintf("builder session %s not found", e.SessionID)
	}
	return fmt.Sprintf("builder session %s not found in project %s", e.SessionID, e.ProjectID)
}

func (e *BuilderSessionNotFoundError) Unwrap() error {
	return ErrBuilderSessionNotFound
}

type BuilderSessionNotAppendableError struct {
	SessionID string
	Status    entities.BuilderSessionStatus
}

func (e *BuilderSessionNotAppendableError) Error() string {
	return fmt.Sprintf("builder session %s is not appendable in status %s", e.SessionID, e.Status)
}

func (e *BuilderSessionNotAppendableError) Unwrap() error {
	return ErrBuilderSessionNotAppendable
}

type BuilderRunNotFoundError struct {
	ProjectID string
	SessionID string
	RunID     string
}

func (e *BuilderRunNotFoundError) Error() string {
	if e.ProjectID == "" {
		return fmt.Sprintf("builder run %s not found in session %s", e.RunID, e.SessionID)
	}
	return fmt.Sprintf("builder run %s not found in session %s for project %s", e.RunID, e.SessionID, e.ProjectID)
}

func (e *BuilderRunNotFoundError) Unwrap() error {
	return ErrBuilderRunNotFound
}

func CreateBuilderSession(ctx context.Context, projectID, userID string, req *models.CreateBuilderSessionRequest) (*models.BuilderSessionDetailResponse, error) {
	now := time.Now().UTC()
	tx := db.DB.WithContext(ctx)
	var detail *models.BuilderSessionDetailResponse

	session := &entities.BuilderSession{
		Base: entities.Base{
			ID: uuid.New(),
		},
		ProjectID:      projectID,
		BuildEnvID:     req.BuildEnvID,
		Title:          req.Title,
		Status:         entities.BuilderSessionStatusProvisioning,
		CreatedBy:      userID,
		LastActivityAt: now,
	}

	message := &entities.BuilderMessage{
		ID:           uuid.New(),
		SessionID:    session.ID,
		Role:         entities.BuilderMessageRoleUser,
		Content:      req.Prompt,
		MetadataJSON: "",
		CreatedBy:    userID,
	}

	intent := AnalyzeBuilderProjectIntent(req.Prompt)
	run := &entities.BuilderRun{
		ID:                       uuid.New(),
		SessionID:                session.ID,
		TriggerMessageID:         message.ID,
		Status:                   entities.BuilderRunStatusQueued,
		Phase:                    builderRunPhaseRef(entities.BuilderRunPhaseQueued),
		RequestedBy:              userID,
		InstructionSummary:       req.Prompt,
		ProviderScope:            stringPointerOrNil(parseBuilderModelSelectionScope(req.SelectedModelKey)),
		ProviderKey:              stringPointerOrNil(strings.TrimSpace(req.ProviderKey)),
		ModelProfileKey:          stringPointerOrNil(strings.TrimSpace(req.ModelProfileKey)),
		PlannedProjectKind:       stringPointerOrNil(intent.ProjectKind),
		PlannedProjectSummary:    intent.Summary,
		PlannedExecutorPolicyKey: stringPointerOrNil(intent.SuggestedExecutorPolicyKey),
		PlannedImageProfileKey:   stringPointerOrNil(intent.SuggestedImageProfileKey),
		ExecutorPolicyKey:        stringPointerOrNil(strings.TrimSpace(req.ExecutorPolicyKey)),
		ExecutionImageProfileKey: stringPointerOrNil(strings.TrimSpace(req.ExecutionImageProfileKey)),
	}

	err := tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(session).Error; err != nil {
			return err
		}

		if err := tx.Create(message).Error; err != nil {
			return err
		}

		if err := tx.Create(run).Error; err != nil {
			return err
		}

		var detailErr error
		detail, detailErr = getBuilderSessionDetail(tx, projectID, session.ID)
		if detailErr != nil {
			return detailErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	builderWorkerQueuedRunNotifier()
	return detail, nil
}

func AppendBuilderSessionMessage(ctx context.Context, projectID, sessionID, userID string, req *models.AppendBuilderSessionMessageRequest) (*models.BuilderSessionDetailResponse, error) {
	now := time.Now().UTC()
	tx := db.DB.WithContext(ctx)
	var detail *models.BuilderSessionDetailResponse

	message := &entities.BuilderMessage{
		ID:           uuid.New(),
		SessionID:    sessionID,
		Role:         entities.BuilderMessageRoleUser,
		Content:      req.Content,
		MetadataJSON: "",
		CreatedBy:    userID,
	}

	intent := AnalyzeBuilderProjectIntent(req.Content)
	run := &entities.BuilderRun{
		ID:                       uuid.New(),
		SessionID:                sessionID,
		TriggerMessageID:         message.ID,
		Status:                   entities.BuilderRunStatusQueued,
		Phase:                    builderRunPhaseRef(entities.BuilderRunPhaseQueued),
		RequestedBy:              userID,
		InstructionSummary:       req.Content,
		ProviderScope:            stringPointerOrNil(parseBuilderModelSelectionScope(req.SelectedModelKey)),
		ProviderKey:              stringPointerOrNil(strings.TrimSpace(req.ProviderKey)),
		ModelProfileKey:          stringPointerOrNil(strings.TrimSpace(req.ModelProfileKey)),
		PlannedProjectKind:       stringPointerOrNil(intent.ProjectKind),
		PlannedProjectSummary:    intent.Summary,
		PlannedExecutorPolicyKey: stringPointerOrNil(intent.SuggestedExecutorPolicyKey),
		PlannedImageProfileKey:   stringPointerOrNil(intent.SuggestedImageProfileKey),
		ExecutorPolicyKey:        stringPointerOrNil(strings.TrimSpace(req.ExecutorPolicyKey)),
		ExecutionImageProfileKey: stringPointerOrNil(strings.TrimSpace(req.ExecutionImageProfileKey)),
	}

	var session entities.BuilderSession

	err := tx.Transaction(func(tx *gorm.DB) error {
		appendableSession, loadErr := loadBuilderSession(tx, projectID, sessionID)
		if loadErr != nil {
			return loadErr
		}
		if err := validateBuilderSessionAppendable(appendableSession); err != nil {
			return err
		}
		session = *appendableSession
		message.SessionID = session.ID
		run.SessionID = session.ID

		if err := touchAppendableBuilderSession(tx, session.ID, now); err != nil {
			return err
		}

		if err := tx.Create(message).Error; err != nil {
			return err
		}

		if err := tx.Create(run).Error; err != nil {
			return err
		}

		var detailErr error
		detail, detailErr = getBuilderSessionDetail(tx, projectID, session.ID)
		if detailErr != nil {
			return detailErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	builderWorkerQueuedRunNotifier()
	return detail, nil
}

func ListBuilderSessions(ctx context.Context, projectID string) ([]models.BuilderSessionListItem, error) {
	return listBuilderSessions(db.DB.WithContext(ctx), projectID)
}

func listBuilderSessions(tx *gorm.DB, projectID string) ([]models.BuilderSessionListItem, error) {
	var rows []models.BuilderSessionListRow
	if err := builderSessionListQuery(tx, projectID).
		Order("bs.created_at DESC, bs.id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	responses := make([]models.BuilderSessionListItem, 0, len(rows))
	for i := range rows {
		responses = append(responses, toBuilderSessionListItem(&rows[i]))
	}

	return responses, nil
}

func GetBuilderSessionDetail(ctx context.Context, projectID, sessionID string) (*models.BuilderSessionDetailResponse, error) {
	readDB := db.DB.WithContext(ctx)
	var detail *models.BuilderSessionDetailResponse
	err := readDB.Transaction(func(tx *gorm.DB) error {
		var detailErr error
		detail, detailErr = getBuilderSessionDetail(tx, projectID, sessionID)
		return detailErr
	})
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func GetBuilderAvailableModelOptions(ctx context.Context, projectID, userID string) ([]models.BuilderAvailableModelOptionResponse, error) {
	return ListBuilderAvailableModelOptions(projectID, userID)
}

func GetBuilderDefaultModelSelection(ctx context.Context, projectID, userID string) (*models.BuilderModelSelectionResponse, error) {
	return GetBuilderModelSelection(projectID, userID)
}

func GetBuilderRun(ctx context.Context, projectID, sessionID, runID string) (*entities.BuilderRun, error) {
	tx := db.DB.WithContext(ctx)
	session, err := loadBuilderSession(tx, projectID, sessionID)
	if err != nil {
		return nil, err
	}

	var run entities.BuilderRun
	if err := tx.Where("id = ? AND session_id = ?", runID, session.ID).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &BuilderRunNotFoundError{ProjectID: projectID, SessionID: session.ID, RunID: runID}
		}
		return nil, err
	}
	return &run, nil
}

func RequestBuilderSessionRunCancel(ctx context.Context, projectID, sessionID, runID string) (*entities.BuilderRun, error) {
	if _, err := GetBuilderRun(ctx, projectID, sessionID, runID); err != nil {
		return nil, err
	}

	return RequestBuilderRunCancel(ctx, runID)
}

func GetBuilderSessionPreview(ctx context.Context, projectID, sessionID string) (*models.BuilderSessionPreviewResponse, error) {
	detail, err := GetBuilderSessionDetail(ctx, projectID, sessionID)
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Preview == nil {
		return &models.BuilderSessionPreviewResponse{}, nil
	}

	preview := &models.BuilderSessionPreviewResponse{
		Available:         detail.Preview.Available,
		Status:            detail.Preview.Status,
		ResolvedRunID:     detail.Preview.ResolvedRunID,
		PublishedAt:       detail.Preview.PublishedAt,
		CompletedAt:       detail.Preview.CompletedAt,
		OutputRoot:        detail.Preview.OutputRoot,
		DefaultEntryPath:  detail.Preview.DefaultEntryPath,
		DownloadAvailable: detail.Preview.DownloadAvailable,
		PreviewAvailable:  detail.Preview.PreviewAvailable,
		IsStale:           detail.Preview.IsStale,
		NewerRunID:        detail.Preview.NewerRunID,
		NewerRunStatus:    detail.Preview.NewerRunStatus,
	}
	if preview.Available && preview.ResolvedRunID != "" {
		preview.DownloadURL = builderSessionSnapshotDownloadPath(projectID, sessionID, preview.ResolvedRunID)
		if preview.PreviewAvailable {
			preview.PreviewLaunchURL = builderSessionPreviewLaunchPath(projectID, sessionID, preview.ResolvedRunID)
		}
	}

	return preview, nil
}

func LaunchBuilderSessionPreview(ctx context.Context, projectID, sessionID, runID string) (*models.BuilderPreviewLaunchResponse, error) {
	snapshot, err := getBuilderSessionSnapshotForRun(ctx, projectID, sessionID, runID)
	if err != nil {
		return nil, err
	}
	if snapshot.Status != entities.BuilderOutputSnapshotStatusPreviewable {
		return nil, &BuilderRunNotFoundError{ProjectID: projectID, SessionID: sessionID, RunID: runID}
	}

	return &models.BuilderPreviewLaunchResponse{
		FrameURL: builderPreviewFramePath(projectID, sessionID, runID),
	}, nil
}

func DownloadBuilderSessionSnapshot(ctx context.Context, projectID, sessionID, runID string, writer io.Writer) error {
	if writer == nil {
		return errors.New("snapshot archive writer is required")
	}
	snapshot, err := getBuilderSessionSnapshotForRun(ctx, projectID, sessionID, runID)
	if err != nil {
		return err
	}

	return WriteBuilderOutputSnapshotArchive(ctx, snapshot, writer)
}

func ResolveBuilderSessionSnapshot(ctx context.Context, projectID, sessionID, runID string) (*entities.BuilderOutputSnapshot, error) {
	return getBuilderSessionSnapshotForRun(ctx, projectID, sessionID, runID)
}

func ResolveBuilderSessionSnapshotFile(ctx context.Context, projectID, sessionID, runID, assetPath string) (*entities.BuilderOutputSnapshot, *entities.BuilderOutputSnapshotFile, error) {
	snapshot, err := getBuilderSessionSnapshotForRun(ctx, projectID, sessionID, runID)
	if err != nil {
		return nil, nil, err
	}
	if snapshot.Status != entities.BuilderOutputSnapshotStatusPreviewable {
		return nil, nil, &BuilderRunNotFoundError{ProjectID: projectID, SessionID: sessionID, RunID: runID}
	}

	resolvedAssetPath := strings.TrimPrefix(strings.TrimSpace(assetPath), "/")
	if resolvedAssetPath == "" {
		resolvedAssetPath = snapshot.DefaultEntryPath
	} else if snapshot.OutputRoot != "" && !strings.HasPrefix(resolvedAssetPath, snapshot.OutputRoot+"/") {
		resolvedAssetPath = path.Join(snapshot.OutputRoot, resolvedAssetPath)
	}
	snapshotFile, err := GetBuilderOutputSnapshotFile(ctx, snapshot.ID, resolvedAssetPath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, &BuilderRunNotFoundError{ProjectID: projectID, SessionID: sessionID, RunID: runID}
		}
		return nil, nil, err
	}

	return snapshot, snapshotFile, nil
}

func getBuilderSessionDetail(tx *gorm.DB, projectID, sessionID string) (*models.BuilderSessionDetailResponse, error) {
	var row models.BuilderSessionDetailRow
	if err := builderSessionDetailQuery(tx, projectID).
		Where("bs.id = ?", sessionID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &BuilderSessionNotFoundError{ProjectID: projectID, SessionID: sessionID}
		}
		return nil, err
	}

	var messages []entities.BuilderMessage
	if err := tx.Where("session_id = ?", sessionID).Order("created_at ASC, id ASC").Find(&messages).Error; err != nil {
		return nil, err
	}

	var runs []entities.BuilderRun
	if err := tx.Where("session_id = ?", sessionID).Order("created_at ASC, id ASC").Find(&runs).Error; err != nil {
		return nil, err
	}

	messageResponses := make([]models.BuilderMessageResponse, 0, len(messages))
	for i := range messages {
		messageResponses = append(messageResponses, toBuilderMessageResponse(&messages[i]))
	}

	runResponses := make([]models.BuilderRunResponse, 0, len(runs))
	for i := range runs {
		runResponses = append(runResponses, toBuilderRunResponse(&runs[i]))
	}

	previewSummary, err := buildBuilderSessionPreviewSummary(tx, sessionID, &row, runs)
	if err != nil {
		return nil, err
	}

	var workspace *models.BuilderWorkspaceSummaryResponse
	artifactResponses := []models.BuilderArtifactSummaryResponse{}
	artifactQuery := tx.Table("builder_artifacts").
		Select("id, session_id, workspace_id, run_id, kind, path, metadata_json, created_at, updated_at")
	if row.CurrentWorkspaceID != "" {
		workspace = toBuilderWorkspaceSummaryFromDetailRow(&row)
		artifactQuery = artifactQuery.Where("session_id = ? AND workspace_id = ?", sessionID, row.CurrentWorkspaceID)
	} else if row.LatestRunID != "" {
		artifactQuery = artifactQuery.Where("session_id = ? AND run_id = ?", sessionID, row.LatestRunID)
	} else {
		artifactQuery = nil
	}

	if artifactQuery != nil {
		var artifactRows []models.BuilderArtifactSummaryRow
		if err := artifactQuery.Order("created_at DESC, id DESC").Find(&artifactRows).Error; err != nil {
			return nil, err
		}

		artifactResponses = make([]models.BuilderArtifactSummaryResponse, 0, len(artifactRows))
		for i := range artifactRows {
			artifactResponses = append(artifactResponses, toBuilderArtifactSummaryResponse(&artifactRows[i]))
		}
	}

	return &models.BuilderSessionDetailResponse{
		Session:   toBuilderSessionResponse(&row),
		Messages:  messageResponses,
		Runs:      runResponses,
		Workspace: workspace,
		Preview:   previewSummary,
		Artifacts: artifactResponses,
	}, nil
}

func buildBuilderSessionPreviewSummary(tx *gorm.DB, sessionID string, row *models.BuilderSessionDetailRow, runs []entities.BuilderRun) (*models.BuilderPreviewSummaryResponse, error) {
	preview := &models.BuilderPreviewSummaryResponse{
		Available:         false,
		Status:            "unavailable",
		DownloadAvailable: false,
		PreviewAvailable:  false,
		IsStale:           false,
	}

	snapshot, err := getLatestSuccessfulBuilderOutputSnapshot(tx, sessionID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return preview, nil
	}

	preview.Available = true
	preview.Status = string(snapshot.Status)
	preview.ResolvedRunID = snapshot.RunID
	preview.PublishedAt = &snapshot.PublishedAt
	preview.OutputRoot = snapshot.OutputRoot
	preview.DefaultEntryPath = snapshot.DefaultEntryPath
	preview.DownloadAvailable = true
	preview.PreviewAvailable = snapshot.Status == entities.BuilderOutputSnapshotStatusPreviewable

	for i := range runs {
		if runs[i].ID == snapshot.RunID {
			preview.Available = true
			preview.CompletedAt = runs[i].CompletedAt
			if row != nil && row.LatestRunID != "" && row.LatestRunID != snapshot.RunID {
				preview.IsStale = true
				preview.NewerRunID = row.LatestRunID
				preview.NewerRunStatus = row.LatestRunStatus
			}
			return preview, nil
		}
	}

	return &models.BuilderPreviewSummaryResponse{
		Available:         false,
		Status:            "unavailable",
		DownloadAvailable: false,
		PreviewAvailable:  false,
		IsStale:           false,
	}, nil
}

func isBuilderSessionAppendable(status entities.BuilderSessionStatus) bool {
	switch status {
	case entities.BuilderSessionStatusProvisioning, entities.BuilderSessionStatusReady, entities.BuilderSessionStatusRunning:
		return true
	default:
		return false
	}
}

func loadBuilderSession(tx *gorm.DB, projectID, sessionID string) (*entities.BuilderSession, error) {
	var session entities.BuilderSession
	query := tx.Where("id = ?", sessionID)
	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if err := query.First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &BuilderSessionNotFoundError{ProjectID: projectID, SessionID: sessionID}
		}
		return nil, err
	}
	return &session, nil
}

func validateBuilderSessionAppendable(session *entities.BuilderSession) error {
	if !isBuilderSessionAppendable(session.Status) {
		return &BuilderSessionNotAppendableError{SessionID: session.ID, Status: session.Status}
	}
	return nil
}

func touchAppendableBuilderSession(tx *gorm.DB, sessionID string, now time.Time) error {
	result := tx.Model(&entities.BuilderSession{}).
		Where("id = ? AND status IN ?", sessionID, []entities.BuilderSessionStatus{
			entities.BuilderSessionStatusProvisioning,
			entities.BuilderSessionStatusReady,
			entities.BuilderSessionStatusRunning,
		}).
		Updates(map[string]any{
			"last_activity_at": now,
			"updated_at":       now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	session, err := loadBuilderSession(tx, "", sessionID)
	if err != nil {
		return err
	}
	return validateBuilderSessionAppendable(session)
}

func builderRunPhaseRef(phase entities.BuilderRunPhase) *entities.BuilderRunPhase {
	return &phase
}

func builderRunPhaseValue(phase *entities.BuilderRunPhase) string {
	if phase == nil {
		return ""
	}
	return string(*phase)
}

func builderSessionDetailQuery(tx *gorm.DB, projectID string) *gorm.DB {
	return tx.Table("builder_sessions AS bs").
		Select(strings.TrimSpace(`
			bs.id AS id,
			bs.project_id AS project_id,
			bs.build_env_id AS build_env_id,
			bs.title AS title,
			COALESCE(bs.summary, '') AS summary,
			bs.status AS status,
			bs.created_by AS created_by,
			bs.created_at AS created_at,
			bs.updated_at AS updated_at,
			bs.last_activity_at AS last_activity_at,
			bs.expires_at AS expires_at,
			COALESCE(lr.id, '') AS latest_run_id,
			COALESCE(lr.status, '') AS latest_run_status,
			COALESCE(cw.id, '') AS current_workspace_id,
			COALESCE(cw.build_env_id, '') AS current_workspace_build_env_id,
			COALESCE(cw.cluster_id, '') AS current_workspace_cluster_id,
			COALESCE(cw.namespace, '') AS current_workspace_namespace,
			COALESCE(cw.pod_name, '') AS current_workspace_pod_name,
			COALESCE(cw.container_name, '') AS current_workspace_container_name,
			COALESCE(cw.status, '') AS current_workspace_status,
			COALESCE(cw.workspace_root, '') AS current_workspace_root,
			cw.terminated_at AS current_workspace_terminated_at,
			cw.created_at AS current_workspace_created_at,
			cw.updated_at AS current_workspace_updated_at
		`)).
		Joins(`LEFT JOIN builder_runs AS lr ON lr.id = (
			SELECT br.id
			FROM builder_runs AS br
			WHERE br.session_id = bs.id
			ORDER BY br.created_at DESC, br.id DESC
			LIMIT 1
		)`).
		Joins(`LEFT JOIN builder_workspaces AS cw ON cw.id = (
			SELECT bw.id
			FROM builder_workspaces AS bw
			WHERE bw.session_id = bs.id AND bw.terminated_at IS NULL
			ORDER BY bw.created_at DESC, bw.id DESC
			LIMIT 1
		)`).
		Where("bs.project_id = ?", projectID)
}

func builderSessionListQuery(tx *gorm.DB, projectID string) *gorm.DB {
	return tx.Table("builder_sessions AS bs").
		Select(strings.TrimSpace(`
			bs.id AS id,
			bs.project_id AS project_id,
			bs.build_env_id AS build_env_id,
			bs.title AS title,
			COALESCE(bs.summary, '') AS summary,
			bs.status AS status,
			bs.created_by AS created_by,
			bs.created_at AS created_at,
			bs.updated_at AS updated_at,
			bs.last_activity_at AS last_activity_at,
			bs.expires_at AS expires_at,
			COALESCE(lr.id, '') AS latest_run_id,
			COALESCE(lr.status, '') AS latest_run_status,
			COALESCE(cw.id, '') AS current_workspace_id,
			COALESCE(cw.status, '') AS current_workspace_status,
			COALESCE(cw.workspace_root, '') AS current_workspace_root,
			COALESCE((
				SELECT COUNT(1)
				FROM builder_artifacts AS ba
				WHERE ba.session_id = bs.id AND ba.workspace_id = cw.id
			), 0) AS artifact_count
		`)).
		Joins(`LEFT JOIN builder_runs AS lr ON lr.id = (
			SELECT br.id
			FROM builder_runs AS br
			WHERE br.session_id = bs.id
			ORDER BY br.created_at DESC, br.id DESC
			LIMIT 1
		)`).
		Joins(`LEFT JOIN builder_workspaces AS cw ON cw.id = (
			SELECT bw.id
			FROM builder_workspaces AS bw
			WHERE bw.session_id = bs.id AND bw.terminated_at IS NULL
			ORDER BY bw.created_at DESC, bw.id DESC
			LIMIT 1
		)`).
		Where("bs.project_id = ?", projectID)
}

func toBuilderSessionResponse(row *models.BuilderSessionDetailRow) models.BuilderSessionResponse {
	return models.BuilderSessionResponse{
		ID:                     row.ID,
		ProjectID:              row.ProjectID,
		BuildEnvID:             row.BuildEnvID,
		Title:                  row.Title,
		Summary:                row.Summary,
		Status:                 row.Status,
		CreatedBy:              row.CreatedBy,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		LastActivityAt:         row.LastActivityAt,
		ExpiresAt:              row.ExpiresAt,
		LatestRunID:            row.LatestRunID,
		LatestRunStatus:        row.LatestRunStatus,
		CurrentWorkspaceID:     row.CurrentWorkspaceID,
		CurrentWorkspaceStatus: row.CurrentWorkspaceStatus,
		CurrentWorkspaceRoot:   row.CurrentWorkspaceRoot,
	}
}

func toBuilderSessionListItem(row *models.BuilderSessionListRow) models.BuilderSessionListItem {
	return models.BuilderSessionListItem{
		ID:                     row.ID,
		ProjectID:              row.ProjectID,
		BuildEnvID:             row.BuildEnvID,
		Title:                  row.Title,
		Summary:                row.Summary,
		Status:                 row.Status,
		CreatedBy:              row.CreatedBy,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		LastActivityAt:         row.LastActivityAt,
		ExpiresAt:              row.ExpiresAt,
		LatestRunID:            row.LatestRunID,
		LatestRunStatus:        row.LatestRunStatus,
		CurrentWorkspaceID:     row.CurrentWorkspaceID,
		CurrentWorkspaceStatus: row.CurrentWorkspaceStatus,
		CurrentWorkspaceRoot:   row.CurrentWorkspaceRoot,
		ArtifactCount:          row.ArtifactCount,
	}
}

func toBuilderMessageResponse(message *entities.BuilderMessage) models.BuilderMessageResponse {
	return models.BuilderMessageResponse{
		ID:           message.ID,
		SessionID:    message.SessionID,
		RunID:        stringPointerValue(message.RunID),
		Role:         string(message.Role),
		Content:      message.Content,
		MetadataJSON: message.MetadataJSON,
		CreatedBy:    message.CreatedBy,
		CreatedAt:    message.CreatedAt,
		UpdatedAt:    message.UpdatedAt,
	}
}

func toBuilderRunResponse(run *entities.BuilderRun) models.BuilderRunResponse {
	return models.BuilderRunResponse{
		ID:                       run.ID,
		SessionID:                run.SessionID,
		TriggerMessageID:         run.TriggerMessageID,
		WorkspaceID:              stringPointerValue(run.WorkspaceID),
		Status:                   string(run.Status),
		Phase:                    builderRunPhaseValue(run.Phase),
		RequestedBy:              run.RequestedBy,
		PlannedProjectKind:       stringPointerValue(run.PlannedProjectKind),
		PlannedProjectSummary:    run.PlannedProjectSummary,
		PlannedExecutorPolicyKey: stringPointerValue(run.PlannedExecutorPolicyKey),
		PlannedImageProfileKey:   stringPointerValue(run.PlannedImageProfileKey),
		ExecutorPolicyKey:        stringPointerValue(run.ExecutorPolicyKey),
		ExecutionImageProfileKey: stringPointerValue(run.ExecutionImageProfileKey),
		ExecutionImageRef:        stringPointerValue(run.ExecutionImageRef),
		ErrorCode:                stringPointerValue(run.ErrorCode),
		ErrorClass:               stringPointerValue(run.ErrorClass),
		InstructionSummary:       run.InstructionSummary,
		ExecutionLog:             run.ExecutionLog,
		StartedAt:                run.StartedAt,
		CompletedAt:              run.CompletedAt,
		ErrorMessage:             run.ErrorMessage,
		CreatedAt:                run.CreatedAt,
		UpdatedAt:                run.UpdatedAt,
	}
}

func toBuilderWorkspaceSummaryFromDetailRow(row *models.BuilderSessionDetailRow) *models.BuilderWorkspaceSummaryResponse {
	createdAt := time.Time{}
	if row.CurrentWorkspaceCreatedAt != nil {
		createdAt = *row.CurrentWorkspaceCreatedAt
	}
	updatedAt := time.Time{}
	if row.CurrentWorkspaceUpdatedAt != nil {
		updatedAt = *row.CurrentWorkspaceUpdatedAt
	}

	return &models.BuilderWorkspaceSummaryResponse{
		ID:            row.CurrentWorkspaceID,
		SessionID:     row.ID,
		BuildEnvID:    row.CurrentWorkspaceBuildEnvID,
		ClusterID:     row.CurrentWorkspaceClusterID,
		Namespace:     row.CurrentWorkspaceNamespace,
		PodName:       row.CurrentWorkspacePodName,
		ContainerName: row.CurrentWorkspaceContainerName,
		Status:        row.CurrentWorkspaceStatus,
		WorkspaceRoot: row.CurrentWorkspaceRoot,
		TerminatedAt:  row.CurrentWorkspaceTerminatedAt,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func toBuilderArtifactSummaryResponse(row *models.BuilderArtifactSummaryRow) models.BuilderArtifactSummaryResponse {
	return models.BuilderArtifactSummaryResponse{
		ID:           row.ID,
		SessionID:    row.SessionID,
		WorkspaceID:  row.WorkspaceID,
		RunID:        row.RunID,
		Kind:         row.Kind,
		Path:         row.Path,
		MetadataJSON: row.MetadataJSON,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPointerOrNil(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func builderSessionSnapshotDownloadPath(projectID, sessionID, runID string) string {
	return fmt.Sprintf("/api/v1/projects/%s/builder-sessions/%s/runs/%s/delivery/download", projectID, sessionID, runID)
}

func builderSessionPreviewLaunchPath(projectID, sessionID, runID string) string {
	return fmt.Sprintf("/api/v1/projects/%s/builder-sessions/%s/runs/%s/preview/launch", projectID, sessionID, runID)
}

func builderPreviewFramePath(projectID, sessionID, runID string) string {
	return fmt.Sprintf("/builder-preview/projects/%s/sessions/%s/runs/%s/", projectID, sessionID, runID)
}

func getBuilderSessionSnapshotForRun(ctx context.Context, projectID, sessionID, runID string) (*entities.BuilderOutputSnapshot, error) {
	if _, err := GetBuilderRun(ctx, projectID, sessionID, runID); err != nil {
		return nil, err
	}

	snapshot, err := GetBuilderOutputSnapshotByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil || snapshot.SessionID != sessionID {
		return nil, &BuilderRunNotFoundError{ProjectID: projectID, SessionID: sessionID, RunID: runID}
	}

	return snapshot, nil
}
